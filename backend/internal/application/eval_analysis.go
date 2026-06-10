package application

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// AnalysisItem 表示某条样本的逐题分析结果。
type AnalysisItem struct {
	Index           int      `json:"index"`
	Prompt          string   `json:"prompt"`
	OriginPrompt    string   `json:"originPrompt,omitempty"`
	Prediction      string   `json:"prediction"`
	Reference       string   `json:"reference"`
	ReferenceTokens []string `json:"referenceTokens,omitempty"`
	HitTokens       []string `json:"hitTokens,omitempty"`
	MissedTokens    []string `json:"missedTokens,omitempty"`
	Score           float64  `json:"score"`    // 0~1
	Category        string   `json:"category"` // failed / low / mid / pass
	Note            string   `json:"note,omitempty"`
}

// AnalysisSummary 分类统计。
type AnalysisSummary struct {
	Total  int `json:"total"`
	Failed int `json:"failed"`
	Low    int `json:"low"`
	Mid    int `json:"mid"`
	Pass   int `json:"pass"`
}

// AnalysisData 是 GET /eval/tasks/:id/analysis 返回的整体结构。
type AnalysisData struct {
	EvalTaskID     string          `json:"evalTaskId"`
	ResultPath     string          `json:"resultPath,omitempty"`
	PredictionPath string          `json:"predictionPath,omitempty"`
	Summary        AnalysisSummary `json:"summary"`
	Items          []AnalysisItem  `json:"items"`
}

// 分类阈值（基于关键词命中率或精确匹配率）。
const (
	scoreLowThreshold  = 0.30
	scorePassThreshold = 0.80
)

// 失败标记：模型调用层错误，按 failed 处理（不同于得分低）。
var predictionFailureMarkers = []string{
	"[Agent error",
	"API returned unexpected status code",
	"Authentication Fails",
	"LLM调用失败",
	"Read timed out",
	"ReadTimeout",
	"Connection error",
}

// GetAnalysis 返回指定任务的逐题分析数据。
// 数据源优先级：results/<role>/<dataset>.json (含 details) > predictions/<role>/<dataset>.json。
// 仅在任务终态产物存在时可用。
func (s *EvalService) GetAnalysis(ctx context.Context, evalTaskID string, userID int64) (*AnalysisData, error) {
	task, err := s.taskRepo.GetByID(ctx, evalTaskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("任务不存在")
	}
	if userID > 0 && task.UserID != userID {
		return nil, fmt.Errorf("任务不存在")
	}
	if strings.TrimSpace(task.OutputDir) == "" {
		return nil, fmt.Errorf("任务尚未产生输出目录")
	}

	runDir, err := pickLatestRunDir(task.OutputDir)
	if err != nil {
		return nil, err
	}

	resultsDir := filepath.Join(task.OutputDir, runDir, "results")
	predictionsDir := filepath.Join(task.OutputDir, runDir, "predictions")

	resultFile := pickFirstJSONFile(resultsDir)
	predictionFile := ""
	if resultFile != "" {
		predictionFile = matchPredictionFile(predictionsDir, resultFile)
	}
	if predictionFile == "" {
		predictionFile = pickFirstJSONFile(predictionsDir)
	}

	resultDetails := map[string]map[string]interface{}{}
	if resultFile != "" {
		if rs, err := loadResultDetails(resultFile); err == nil {
			resultDetails = rs
		}
	}

	predictionEntries := map[string]map[string]interface{}{}
	if predictionFile != "" {
		if ps, err := loadPredictionEntries(predictionFile); err == nil {
			predictionEntries = ps
		}
	}

	if len(resultDetails) == 0 && len(predictionEntries) == 0 {
		return nil, fmt.Errorf("未找到可用的逐题数据：%s 或 %s", resultsDir, predictionsDir)
	}

	keys := mergeIndexKeys(resultDetails, predictionEntries)

	items := make([]AnalysisItem, 0, len(keys))
	for _, k := range keys {
		item := buildAnalysisItem(k, resultDetails[k], predictionEntries[k])
		items = append(items, item)
	}

	summary := AnalysisSummary{Total: len(items)}
	for _, it := range items {
		switch it.Category {
		case "failed":
			summary.Failed++
		case "low":
			summary.Low++
		case "mid":
			summary.Mid++
		case "pass":
			summary.Pass++
		}
	}

	return &AnalysisData{
		EvalTaskID:     evalTaskID,
		ResultPath:     resultFile,
		PredictionPath: predictionFile,
		Summary:        summary,
		Items:          items,
	}, nil
}

// pickFirstJSONFile 在 dir 下递归找第一份 .json（优先 role 子目录第一份）。
func pickFirstJSONFile(dir string) string {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return ""
	}
	candidates := []string{}
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			candidates = append(candidates, p)
		}
		return nil
	})
	if len(candidates) == 0 {
		return ""
	}
	sort.Strings(candidates)
	return candidates[0]
}

// matchPredictionFile 根据 result 文件路径在 predictionsDir 找同名 .json。
func matchPredictionFile(predictionsDir string, resultFile string) string {
	if predictionsDir == "" || resultFile == "" {
		return ""
	}
	base := filepath.Base(resultFile)
	role := filepath.Base(filepath.Dir(resultFile))
	guess := filepath.Join(predictionsDir, role, base)
	if st, err := os.Stat(guess); err == nil && !st.IsDir() {
		return guess
	}
	// 任意 role 下同名文件
	matches, _ := filepath.Glob(filepath.Join(predictionsDir, "*", base))
	for _, m := range matches {
		if st, err := os.Stat(m); err == nil && !st.IsDir() {
			return m
		}
	}
	return ""
}

// loadResultDetails 读取 results/<role>/<file>.json 的 details 字典。
// details 中除 "type" 之外的其他字符串数字键即为逐题记录。
func loadResultDetails(path string) (map[string]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := map[string]map[string]interface{}{}
	details, ok := raw["details"].(map[string]interface{})
	if !ok {
		return out, nil
	}
	for k, v := range details {
		if k == "type" {
			continue
		}
		if m, ok := v.(map[string]interface{}); ok {
			out[k] = m
		}
	}
	return out, nil
}

// loadPredictionEntries 读取 predictions/<role>/<file>.json：顶层数字键 -> {origin_prompt, prediction, gold}。
func loadPredictionEntries(path string) (map[string]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := map[string]map[string]interface{}{}
	for k, v := range raw {
		if m, ok := v.(map[string]interface{}); ok {
			out[k] = m
		}
	}
	return out, nil
}

// mergeIndexKeys 合并两个数据源的索引键并按数字顺序排列。
func mergeIndexKeys(a, b map[string]map[string]interface{}) []string {
	set := map[string]struct{}{}
	for k := range a {
		set[k] = struct{}{}
	}
	for k := range b {
		set[k] = struct{}{}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ai, aerr := strconv.Atoi(keys[i])
		bj, berr := strconv.Atoi(keys[j])
		if aerr == nil && berr == nil {
			return ai < bj
		}
		return keys[i] < keys[j]
	})
	return keys
}

func buildAnalysisItem(key string, detail map[string]interface{}, prediction map[string]interface{}) AnalysisItem {
	idx, _ := strconv.Atoi(key)
	item := AnalysisItem{Index: idx}

	// prompt：优先 details.prompt，回退 predictions.origin_prompt
	item.Prompt = stringify(firstNonEmpty(getField(detail, "prompt"), getField(prediction, "origin_prompt")))
	item.OriginPrompt = stringify(getField(prediction, "origin_prompt"))

	// prediction：优先 details.predictions / origin_prediction，回退 predictions.prediction
	predRaw := firstNonEmpty(
		getField(detail, "predictions"),
		getField(detail, "origin_prediction"),
		getField(prediction, "prediction"),
	)
	item.Prediction = stringify(predRaw)

	// reference / gold
	refRaw := firstNonEmpty(getField(detail, "references"), getField(prediction, "gold"))
	item.Reference = stringify(refRaw)
	item.ReferenceTokens = extractTokens(refRaw)

	// 评分与分类
	if isPredictionFailure(item.Prediction) {
		item.Category = "failed"
		item.Score = 0
		item.Note = "模型调用失败"
		return item
	}

	score, hits, missed, note := scorePrediction(item.Prediction, item.ReferenceTokens, refRaw)
	item.Score = score
	item.HitTokens = hits
	item.MissedTokens = missed
	item.Note = note
	item.Category = classifyScore(score)
	return item
}

// firstNonEmpty 返回第一个非空字段。
func firstNonEmpty(values ...interface{}) interface{} {
	for _, v := range values {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok {
			if strings.TrimSpace(s) != "" {
				return v
			}
			continue
		}
		// 非字符串（list / dict）直接返回
		return v
	}
	return nil
}

func getField(m map[string]interface{}, key string) interface{} {
	if m == nil {
		return nil
	}
	return m[key]
}

// stringify 把任意值转为可显示字符串。
// list/dict 会序列化为 JSON。
func stringify(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%v", x)
		}
		return string(b)
	}
}

// extractTokens 从 reference 中尝试解析关键词列表。
// 支持：
//   - []interface{}：直接转 string 列表
//   - 字符串内 JSON 数组
//   - 字符串内"['a', 'b']"伪 Python 列表
//   - 否则按整段字符串作单 token
func extractTokens(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, e := range x {
			s := strings.TrimSpace(stringify(e))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		// 尝试 JSON
		if strings.HasPrefix(s, "[") {
			var arr []interface{}
			if err := json.Unmarshal([]byte(s), &arr); err == nil {
				return extractTokens(arr)
			}
			// Python 风格：把单引号转成双引号再试
			if quoted := strings.ReplaceAll(s, "'", "\""); quoted != s {
				if err := json.Unmarshal([]byte(quoted), &arr); err == nil {
					return extractTokens(arr)
				}
			}
		}
		return []string{s}
	default:
		return []string{stringify(v)}
	}
}

// isPredictionFailure 判断是否属于"模型调用失败"。
func isPredictionFailure(prediction string) bool {
	p := strings.TrimSpace(prediction)
	if p == "" {
		return true
	}
	for _, marker := range predictionFailureMarkers {
		if strings.Contains(p, marker) {
			return true
		}
	}
	return false
}

// scorePrediction 对一条预测打分。
//
//	关键词列表：命中关键词数 / 总数（区分大小写不敏感的子串包含）。
//	单字符串：完全匹配则 1.0，否则 0.0。
func scorePrediction(prediction string, tokens []string, refRaw interface{}) (float64, []string, []string, string) {
	pred := strings.ToLower(prediction)
	if len(tokens) == 0 {
		return 0, nil, nil, "缺少参考答案"
	}
	// 整段字符串型：单 token 且不像关键词列表（含空白），直接做精确/相似匹配
	if len(tokens) == 1 {
		// 如果原始数据是 list 但只有一个元素，仍按关键词模式
		if _, isList := refRaw.([]interface{}); !isList {
			refStr := strings.TrimSpace(strings.ToLower(tokens[0]))
			if refStr == "" {
				return 0, nil, nil, ""
			}
			if pred == refStr {
				return 1.0, []string{tokens[0]}, nil, ""
			}
			if strings.Contains(pred, refStr) {
				return 1.0, []string{tokens[0]}, nil, ""
			}
			return 0, nil, []string{tokens[0]}, ""
		}
	}
	hits := make([]string, 0, len(tokens))
	missed := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		t := strings.ToLower(strings.TrimSpace(tok))
		if t == "" {
			continue
		}
		if strings.Contains(pred, t) {
			hits = append(hits, tok)
		} else {
			missed = append(missed, tok)
		}
	}
	if len(tokens) == 0 {
		return 0, nil, nil, ""
	}
	score := float64(len(hits)) / float64(len(tokens))
	return score, hits, missed, ""
}

func classifyScore(score float64) string {
	if score >= scorePassThreshold {
		return "pass"
	}
	if score >= scoreLowThreshold {
		return "mid"
	}
	if score > 0 {
		return "low"
	}
	return "low"
}
