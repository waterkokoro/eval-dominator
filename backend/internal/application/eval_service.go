package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"eval-dominator/backend/internal/config"
	"eval-dominator/backend/internal/domain"
	evalv1 "eval-dominator/backend/internal/infrastructure/grpc/gen"
)

type EvalTaskRepository interface {
	Create(ctx context.Context, task domain.EvalTask) error
	GetByID(ctx context.Context, id string) (*domain.EvalTask, error)
	UpdateStatus(ctx context.Context, id string, status domain.EvalTaskStatus, outputDir string, errorCode string, errorMessage string) error
	List(ctx context.Context, query domain.EvalTaskListQuery) ([]domain.EvalTask, int64, error)
}

type EvalResultRepository interface {
	Save(ctx context.Context, result domain.EvalResult) error
	GetByTaskID(ctx context.Context, evalTaskID string) (*domain.EvalResult, error)
}

type ModelRepository interface {
	Save(ctx context.Context, model domain.Model) error
	GetByID(ctx context.Context, id int64) (*domain.Model, error)
}

type CoreClient interface {
	ValidateEvalConfig(ctx context.Context, config *evalv1.EvalConfig) error
	BuildEvalConfig(ctx context.Context, config *evalv1.EvalConfig) (*evalv1.BuildEvalConfigResponse, error)
	ExecuteEval(ctx context.Context, evalTaskID string, config *evalv1.EvalConfig, configPath string, outputDir string, reuseTimestamp string) (*evalv1.ExecuteEvalResponse, error)
	ParseEvalResult(ctx context.Context, evalTaskID string, outputDir string) (*evalv1.EvalResult, error)
	CancelEval(ctx context.Context, evalTaskID string) (bool, error)
}

type DatasetLookup interface {
	GetByCode(ctx context.Context, code string) (*domain.Dataset, error)
}

type EvalService struct {
	taskRepo    EvalTaskRepository
	resultRepo  EvalResultRepository
	modelRepo   ModelRepository
	datasetRepo DatasetLookup
	coreClient  CoreClient
	config      config.EvalConfig

	// 用户主动 cancel 的任务集合：goroutine 在 Core 报错回来时若发现自己被 cancel 了，
	// 不再覆盖写 failed，保持 cancelled 状态。in-memory 即可（崩溃后由 stale 任务的 timeout/手动重置兜底）。
	cancelMu        sync.Mutex
	cancelRequested map[string]struct{}
}

type CreateEvalTaskInput struct {
	TaskName      string
	UserID        int64
	Provider      string
	ModelName     string
	DisplayName   string
	Version       string
	BaseURL       string
	APIKey        string
	ModelPresetID int64
	DatasetType   string
	DatasetName   string
	EvaluatorType string // rouge / accuracy / keyword_match / em / bleu / jieba_rouge（默认 rouge）
	SaveModel     bool
	Params        map[string]string
	Runtime       RuntimeInput
}

// RuntimeInput 前端传入的运行时参数，0 值表示使用服务端默认值。
type RuntimeInput struct {
	TimeoutSeconds int
	MaxWorkers     int
	KeepRawOutputs bool
	HasValues      bool // true 表示前端明确传入了 runtime 参数
}

func NewEvalService(taskRepo EvalTaskRepository, resultRepo EvalResultRepository, modelRepo ModelRepository, datasetRepo DatasetLookup, coreClient CoreClient, cfg config.EvalConfig) *EvalService {
	return &EvalService{
		taskRepo:        taskRepo,
		resultRepo:      resultRepo,
		modelRepo:       modelRepo,
		datasetRepo:     datasetRepo,
		coreClient:      coreClient,
		config:          cfg,
		cancelRequested: map[string]struct{}{},
	}
}

func (s *EvalService) CreateTask(ctx context.Context, input CreateEvalTaskInput) (*domain.EvalTask, error) {
	if input.ModelPresetID > 0 {
		preset, err := s.modelRepo.GetByID(ctx, input.ModelPresetID)
		if err != nil {
			return nil, fmt.Errorf("模型预设不存在或不可用")
		}
		if preset.UserID != input.UserID {
			return nil, fmt.Errorf("模型预设不存在或不可用")
		}
		input.APIKey = preset.APIKey
		input.Provider = preset.Provider
		input.ModelName = preset.ModelName
		if input.BaseURL == "" {
			input.BaseURL = preset.BaseURL
		}
		if input.DisplayName == "" {
			input.DisplayName = preset.DisplayName
		}
		if input.Version == "" {
			input.Version = preset.Version
		}
		input.SaveModel = false
	}

	if input.Provider == "" {
		return nil, fmt.Errorf("模型服务商不能为空")
	}
	if input.ModelName == "" {
		return nil, fmt.Errorf("模型名称不能为空")
	}
	if input.APIKey == "" {
		return nil, fmt.Errorf("API Key 不能为空")
	}
	if input.DatasetType == "" {
		input.DatasetType = s.config.DefaultDatasetType
	}
	if input.DatasetName == "" {
		input.DatasetName = "demo"
	}

	// 兜底校验：PPL 推理模式只支持本地 HuggingFace 模型，远程 API 模型一律拒绝。
	if s.datasetRepo != nil && input.DatasetName != "" {
		if dataset, err := s.datasetRepo.GetByCode(ctx, input.DatasetName); err == nil && dataset != nil && !dataset.IsRemoteCompatible() {
			return nil, fmt.Errorf("数据集「%s」使用 PPL 推理方式，仅支持本地 HuggingFace 模型，请改选 _gen 后缀的数据集", dataset.DisplayName)
		}
	}

	taskName := normalizeEvalTaskName(input.TaskName)
	task := domain.EvalTask{
		ID:            newEvalTaskID(),
		TaskName:      taskName,
		UserID:        input.UserID,
		ModelProvider: input.Provider,
		ModelName:     input.ModelName,
		ModelBaseURL:  input.BaseURL,
		ModelPresetID: input.ModelPresetID,
		EvaluatorType: input.EvaluatorType,
		DatasetType:   input.DatasetType,
		DatasetName:   input.DatasetName,
		Status:        domain.EvalTaskStatusPending,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}
	if input.SaveModel && s.config.SaveAPIKeyEnabled {
		if err := s.saveModelPreset(ctx, input); err != nil {
			return nil, err
		}
	}
	go s.executeTask(context.Background(), task, input)
	return &task, nil
}

func (s *EvalService) GetTask(ctx context.Context, id string) (*domain.EvalTask, error) {
	return s.taskRepo.GetByID(ctx, id)
}

func (s *EvalService) GetResult(ctx context.Context, evalTaskID string) (*domain.EvalResult, error) {
	return s.resultRepo.GetByTaskID(ctx, evalTaskID)
}

type EvalTaskListResult struct {
	Items    []domain.EvalTask
	Total    int64
	Page     int
	PageSize int
}

func (s *EvalService) ListTasks(ctx context.Context, query domain.EvalTaskListQuery) (*EvalTaskListResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	tasks, total, err := s.taskRepo.List(ctx, query)
	if err != nil {
		return nil, err
	}
	return &EvalTaskListResult{
		Items:    tasks,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// CancelTask 让 Core 终止任务对应的 OpenCompass 子进程组，并把数据库状态置为 cancelled。
// 已经处于终态的任务直接返回错误。
func (s *EvalService) CancelTask(ctx context.Context, evalTaskID string, userID int64) error {
	task, err := s.taskRepo.GetByID(ctx, evalTaskID)
	if err != nil {
		return err
	}
	if userID > 0 && task.UserID != userID {
		return fmt.Errorf("任务不存在")
	}
	if task.Status.IsTerminal() {
		return fmt.Errorf("任务已结束（status=%s），不可终止", task.Status)
	}

	// 先打 cancel 标记，避免 executeTask 在 Core 报错后覆盖回 failed。
	s.markCancelled(evalTaskID)

	// 通知 Core 杀子进程组（即使没在 running 也无所谓，Core 会返回 running=false）。
	if _, err := s.coreClient.CancelEval(ctx, evalTaskID); err != nil {
		log.Printf("Core CancelEval 失败 task=%s: %v", evalTaskID, err)
	}

	if err := s.taskRepo.UpdateStatus(ctx, evalTaskID, domain.EvalTaskStatusCancelled, task.OutputDir, "CANCELLED", "用户主动终止"); err != nil {
		return fmt.Errorf("写入终止状态失败: %w", err)
	}
	return nil
}

// runDirPattern 匹配 OpenCompass 0.5.x 的运行目录命名 <YYYYMMDD_HHMMSS>。
var runDirPattern = regexp.MustCompile(`^\d{8}_\d{6}$`)

// RerunEvaluateNode 仅重跑 evaluate 节点：复用 task.OutputDir/<latest_ts>/predictions
// 让 OpenCompass 通过 -r 跳过推理，仅重新生成 results / summary。
//
// 校验：
//  1. 任务存在且属于当前用户
//  2. 必须处于终态（succeeded / failed / cancelled / timeout）
//  3. task.OutputDir 非空且实际存在
//  4. OutputDir 下存在 build 阶段生成的 opencompass_eval_config.py
//  5. OutputDir 下能找到最新的 <YYYYMMDD_HHMMSS> 运行子目录
//  6. 该子目录下 predictions/ 存在并至少有一份 *.json 预测文件
//  7. 任务当前不在 in-flight rerun 中（依赖 DB 状态机：上面 1/2 已经把这条限制收敛到终态）
//
// 校验通过后异步执行 ExecuteEval(reuseTimestamp=ts) → ParseEvalResult → saveResult。
func (s *EvalService) RerunEvaluateNode(ctx context.Context, evalTaskID string, userID int64) error {
	task, err := s.taskRepo.GetByID(ctx, evalTaskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("任务不存在")
	}
	if userID > 0 && task.UserID != userID {
		return fmt.Errorf("任务不存在")
	}
	if !task.Status.IsTerminal() {
		return fmt.Errorf("任务尚未结束（status=%s），不能仅重跑评测节点", task.Status)
	}
	if strings.TrimSpace(task.OutputDir) == "" {
		return fmt.Errorf("任务缺少输出目录，无法复用 predictions 重跑评测")
	}
	if st, err := os.Stat(task.OutputDir); err != nil || !st.IsDir() {
		return fmt.Errorf("任务输出目录不存在或已被清理: %s", task.OutputDir)
	}

	configPath := filepath.Join(task.OutputDir, "opencompass_eval_config.py")
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("缺少 build 阶段产物 opencompass_eval_config.py，无法重跑评测")
	}

	runTimestamp, err := pickLatestRunDir(task.OutputDir)
	if err != nil {
		return err
	}
	predictionsDir := filepath.Join(task.OutputDir, runTimestamp, "predictions")
	hasPredictions, err := hasPredictionFile(predictionsDir)
	if err != nil {
		return fmt.Errorf("检查 predictions 失败: %w", err)
	}
	if !hasPredictions {
		return fmt.Errorf("未找到可复用的预测结果（%s 下没有 *.json），请整体重跑任务", predictionsDir)
	}

	// 立刻把状态切回 running，避免前端反复点击；后续异步跑 ExecuteEval。
	// 注意：UpdateStatus 第三个参数为空串时部分实现可能会清空 OutputDir，所以仍传原值。
	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusRunning, task.OutputDir, "", ""); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	taskCopy := *task
	go s.executeRerunEvaluate(context.Background(), taskCopy, configPath, runTimestamp)
	return nil
}

// executeRerunEvaluate 异步执行 evaluate 节点重跑流程，沿用 executeTask 后半段的状态机。
func (s *EvalService) executeRerunEvaluate(ctx context.Context, task domain.EvalTask, configPath string, reuseTimestamp string) {
	// reuse 模式下 Core 不会重新生成 mmengine 配置、不会调用 LLM，
	// 所以 EvalConfig 用最小占位即可：只填 runtime 必需字段和 eval_task_id。
	evalConfig := &evalv1.EvalConfig{
		Model:   &evalv1.ModelConfig{Type: evalv1.ModelType_MODEL_TYPE_REMOTE_API},
		Dataset: &evalv1.DatasetConfig{},
		Runtime: &evalv1.RuntimeConfig{
			WorkDir:        s.config.OutputDir,
			TimeoutSeconds: int32(s.config.DefaultTimeoutSecs),
			MaxWorkers:     1,
			KeepRawOutputs: true,
		},
		ExtraParams: map[string]string{"eval_task_id": task.ID},
	}

	executeResp, err := s.coreClient.ExecuteEval(ctx, task.ID, evalConfig, configPath, task.OutputDir, reuseTimestamp)
	if err != nil {
		if s.consumeCancelled(task.ID) {
			log.Printf("evaluate 重跑被用户终止 task=%s: %v", task.ID, err)
			return
		}
		s.failTask(ctx, task.ID, "RERUN_EVAL_FAILED", err)
		return
	}
	if s.consumeCancelled(task.ID) {
		log.Printf("evaluate 重跑被用户终止 task=%s（ExecuteEval 已退出）", task.ID)
		return
	}

	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusParsing, executeResp.GetOutputDir(), "", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
		return
	}
	result, err := s.coreClient.ParseEvalResult(ctx, task.ID, executeResp.GetOutputDir())
	if err != nil {
		s.failTask(ctx, task.ID, "PARSE_FAILED", err)
		return
	}
	if err := s.saveResult(ctx, task.ID, result); err != nil {
		s.failTask(ctx, task.ID, "SAVE_RESULT_FAILED", err)
		return
	}
	if reason := failureReason(result); reason != "" {
		if updateErr := s.taskRepo.UpdateStatus(ctx, task.ID, domain.EvalTaskStatusFailed, executeResp.GetOutputDir(), "NO_VALID_METRIC", reason); updateErr != nil {
			log.Printf("更新任务状态失败: %v", updateErr)
		}
		return
	}
	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusSucceeded, executeResp.GetOutputDir(), "", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
	}
}

// pickLatestRunDir 在 outputDir 下查找形如 YYYYMMDD_HHMMSS 的最新运行子目录。
func pickLatestRunDir(outputDir string) (string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return "", fmt.Errorf("读取输出目录失败: %w", err)
	}
	latest := ""
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !runDirPattern.MatchString(name) {
			continue
		}
		if name > latest {
			latest = name
		}
	}
	if latest == "" {
		return "", fmt.Errorf("未在 %s 下找到 OpenCompass 运行目录（YYYYMMDD_HHMMSS）", outputDir)
	}
	return latest, nil
}

// hasPredictionFile 检查目录下是否至少有一份 *.json 预测文件（递归）。
func hasPredictionFile(dir string) (bool, error) {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return false, nil
	}
	found := false
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil && walkErr != filepath.SkipAll {
		return found, walkErr
	}
	return found, nil
}

func (s *EvalService) markCancelled(evalTaskID string) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	s.cancelRequested[evalTaskID] = struct{}{}
}

func (s *EvalService) consumeCancelled(evalTaskID string) bool {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	if _, ok := s.cancelRequested[evalTaskID]; ok {
		delete(s.cancelRequested, evalTaskID)
		return true
	}
	return false
}

// ArtifactFile 描述要响应给前端的产物文件位置（绝对路径）+ 类型/大小，便于 Handler 决定是预览还是下载。
type ArtifactFile struct {
	AbsolutePath string
	RelativePath string
	Size         int64
	IsText       bool
	ContentType  string
}

const (
	maxPreviewBytes = 512 * 1024 // 预览最多 512 KB
)

var textArtifactExt = map[string]struct{}{
	".log":   {},
	".out":   {},
	".txt":   {},
	".md":    {},
	".csv":   {},
	".json":  {},
	".jsonl": {},
	".yaml":  {},
	".yml":   {},
	".py":    {},
}

// ResolveArtifactFile 校验路径合法（必须落在 OutputDir 下）后返回文件元信息。
// 同时禁止符号链接逃逸。
func (s *EvalService) ResolveArtifactFile(ctx context.Context, evalTaskID string, requestPath string) (*ArtifactFile, error) {
	if evalTaskID == "" {
		return nil, fmt.Errorf("任务 ID 为空")
	}
	if strings.TrimSpace(requestPath) == "" {
		return nil, fmt.Errorf("文件路径为空")
	}

	root, err := filepath.Abs(s.config.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("解析输出根目录失败: %w", err)
	}

	abs, err := filepath.Abs(requestPath)
	if err != nil {
		return nil, fmt.Errorf("解析路径失败: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// 文件可能尚未生成，这种情况直接报 404；但仍要先校验目录前缀，防止 traversal。
		resolved = abs
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolvedRoot = root
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return nil, fmt.Errorf("路径越界，必须位于 runtime 输出目录下")
	}
	// 还要确保和当前任务相关：第一段必须是 evalTaskID 或者是任务 OutputDir 的前缀。
	first := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
	if first != evalTaskID {
		// 任务的 OutputDir 不一定就在 OutputDir/{evalTaskID}（旧任务可能直接落在 OutputDir 里）。
		task, err := s.taskRepo.GetByID(ctx, evalTaskID)
		if err != nil || task == nil {
			return nil, fmt.Errorf("任务不存在")
		}
		if task.OutputDir == "" || !strings.HasPrefix(resolved, task.OutputDir) {
			return nil, fmt.Errorf("路径越界，与该任务无关")
		}
	}

	stat, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("文件不存在或不可读")
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("不允许读取目录")
	}

	ext := strings.ToLower(filepath.Ext(resolved))
	_, isText := textArtifactExt[ext]
	contentType := "application/octet-stream"
	switch ext {
	case ".json", ".jsonl":
		contentType = "application/json; charset=utf-8"
	case ".csv":
		contentType = "text/csv; charset=utf-8"
	case ".md", ".txt", ".log", ".out", ".py", ".yaml", ".yml":
		contentType = "text/plain; charset=utf-8"
	}

	return &ArtifactFile{
		AbsolutePath: resolved,
		RelativePath: rel,
		Size:         stat.Size(),
		IsText:       isText,
		ContentType:  contentType,
	}, nil
}

// PreviewArtifactFile 读取文本类产物的前 N KB 用于在线预览。
func (s *EvalService) PreviewArtifactFile(ctx context.Context, evalTaskID string, requestPath string) (*ArtifactFile, string, bool, error) {
	file, err := s.ResolveArtifactFile(ctx, evalTaskID, requestPath)
	if err != nil {
		return nil, "", false, err
	}
	if !file.IsText {
		return file, "", false, nil
	}
	f, err := os.Open(file.AbsolutePath)
	if err != nil {
		return file, "", false, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()
	buf := make([]byte, maxPreviewBytes)
	n, _ := f.Read(buf)
	truncated := file.Size > int64(n)
	return file, string(buf[:n]), truncated, nil
}

// LogFileMeta 描述一个可供前端查看的日志文件元信息。
type LogFileMeta struct {
	ID          string `json:"id"`          // 唯一标识（基于路径 hash 或类型+名字）
	DisplayName string `json:"displayName"` // 展示名（如 "推理: piqa"）
	Type        string `json:"type"`        // main / infer / eval / system
	Path        string `json:"path"`        // 绝对路径（仅后端使用，可选返回）
	ModTime     int64  `json:"mtime"`       // mtime 毫秒
	Size        int64  `json:"size"`
}

// GetTaskLog 读取指定任务的日志内容。
// 当 logID 非空时，按 logID 解析对应日志文件；为空则按"最近更新"的日志兜底。
func (s *EvalService) GetTaskLog(ctx context.Context, evalTaskID string, logID string, tail int) (string, error) {
	if tail <= 0 {
		tail = 200
	}

	target := ""
	if logID != "" {
		// 通过 logID 反查具体文件路径
		logs := s.ListTaskLogs(ctx, evalTaskID)
		for _, lg := range logs {
			if lg.ID == logID {
				target = lg.Path
				break
			}
		}
	}
	if target == "" {
		// 兜底：选最近更新的日志（保持旧行为）
		candidates := s.candidateLogPaths(ctx, evalTaskID)
		target = pickFreshestExisting(candidates)
	}
	if target == "" {
		return "", nil
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("读取日志失败: %w", err)
	}
	return tailLines(string(data), tail), nil
}

// ListTaskLogs 列举该任务所有可用的日志文件，按类型分组、按 mtime 倒序。
// 类型：main（任务主日志）/ infer（推理子任务）/ eval（评测子任务）/ system（core / backend）。
func (s *EvalService) ListTaskLogs(ctx context.Context, evalTaskID string) []LogFileMeta {
	out := make([]LogFileMeta, 0, 16)
	seen := map[string]struct{}{}

	add := func(meta LogFileMeta) {
		if meta.Path == "" {
			return
		}
		if _, ok := seen[meta.Path]; ok {
			return
		}
		st, err := os.Stat(meta.Path)
		if err != nil || st.IsDir() {
			return
		}
		seen[meta.Path] = struct{}{}
		meta.ModTime = st.ModTime().UnixMilli()
		meta.Size = st.Size()
		if meta.ID == "" {
			meta.ID = makeLogID(meta.Type, meta.Path)
		}
		out = append(out, meta)
	}

	root := s.config.OutputDir
	taskRoot := ""
	if root != "" {
		taskRoot = filepath.Join(root, evalTaskID)
		add(LogFileMeta{
			Type:        "main",
			Path:        filepath.Join(taskRoot, "opencompass.log"),
			DisplayName: "opencompass.log",
		})
		base := filepath.Dir(root)
		add(LogFileMeta{
			Type:        "system",
			Path:        filepath.Join(base, "logs", "core", evalTaskID+".log"),
			DisplayName: "core",
		})
		add(LogFileMeta{
			Type:        "system",
			Path:        filepath.Join(base, "logs", "backend", evalTaskID+".log"),
			DisplayName: "backend",
		})
	}

	// DB 兜底
	if task, err := s.taskRepo.GetByID(ctx, evalTaskID); err == nil && task != nil && task.OutputDir != "" {
		add(LogFileMeta{
			Type:        "main",
			Path:        filepath.Join(task.OutputDir, "opencompass.log"),
			DisplayName: "opencompass.log",
		})
		if taskRoot == "" {
			taskRoot = task.OutputDir
		}
	}

	// 子任务日志：<task_root>/<timestamp>/logs/{infer,eval}/<model>/<dataset>.out
	if taskRoot != "" {
		matches, _ := filepath.Glob(filepath.Join(taskRoot, "*", "logs", "infer", "*", "*.out"))
		for _, p := range matches {
			add(LogFileMeta{
				Type:        "infer",
				Path:        p,
				DisplayName: subtaskLogDisplayName(p),
			})
		}
		matches, _ = filepath.Glob(filepath.Join(taskRoot, "*", "logs", "eval", "*", "*.out"))
		for _, p := range matches {
			add(LogFileMeta{
				Type:        "eval",
				Path:        p,
				DisplayName: subtaskLogDisplayName(p),
			})
		}
	}

	return out
}

// subtaskLogDisplayName 从形如 .../logs/infer/<model>/<dataset>.out 中提取 "<dataset>"。
func subtaskLogDisplayName(p string) string {
	base := filepath.Base(p)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return name
}

// makeLogID 用 type + 路径相对最末两段做稳定 id，前端把它当作不透明字符串。
func makeLogID(typ string, p string) string {
	clean := strings.ReplaceAll(p, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	clean = strings.ReplaceAll(clean, ":", "_")
	if len(clean) > 80 {
		clean = clean[len(clean)-80:]
	}
	return typ + "-" + clean
}

func (s *EvalService) candidateLogPaths(ctx context.Context, evalTaskID string) []string {
	candidates := make([]string, 0, 16)

	if result, err := s.resultRepo.GetByTaskID(ctx, evalTaskID); err == nil && result != nil && result.LogPath != "" {
		candidates = append(candidates, result.LogPath)
	}

	root := s.config.OutputDir
	taskRoot := ""
	if root != "" {
		taskRoot = filepath.Join(root, evalTaskID)
		// 顶层主日志（execute_eval 写入的合并 stdout）
		candidates = append(candidates, filepath.Join(taskRoot, "opencompass.log"))
		base := filepath.Dir(root)
		candidates = append(candidates,
			filepath.Join(base, "logs", "core", evalTaskID+".log"),
			filepath.Join(base, "logs", "backend", evalTaskID+".log"),
		)
	}

	// 兜底：DB 里 task.OutputDir 可能与配置里的 OutputDir 不一致（旧任务）
	if task, err := s.taskRepo.GetByID(ctx, evalTaskID); err == nil && task != nil && task.OutputDir != "" {
		candidates = append(candidates, filepath.Join(task.OutputDir, "opencompass.log"))
		if taskRoot == "" {
			taskRoot = task.OutputDir
		}
	}

	// OpenCompass 真正的子任务日志在 <task_root>/<timestamp>/logs/{infer,eval}/<model>/<dataset>.out
	// 对运行中的任务来说，这是唯一会“动”的文件。把所有匹配到的 .out 也加入候选，
	// pickFreshestExisting 会按 mtime 选最新的。
	if taskRoot != "" {
		matches, _ := filepath.Glob(filepath.Join(taskRoot, "*", "logs", "infer", "*", "*.out"))
		candidates = append(candidates, matches...)
		matches, _ = filepath.Glob(filepath.Join(taskRoot, "*", "logs", "eval", "*", "*.out"))
		candidates = append(candidates, matches...)
	}

	return candidates
}

// pickFreshestExisting 返回 candidates 里 mtime 最新的存在文件路径；都不存在返回空串。
func pickFreshestExisting(candidates []string) string {
	bestPath := ""
	var bestMtime time.Time
	seen := map[string]struct{}{}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		st, err := os.Stat(p)
		if err != nil || st.IsDir() || st.Size() == 0 {
			continue
		}
		if bestPath == "" || st.ModTime().After(bestMtime) {
			bestPath = p
			bestMtime = st.ModTime()
		}
	}
	return bestPath
}

func tailLines(content string, tail int) string {
	if tail <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= tail {
		return content
	}
	return strings.Join(lines[len(lines)-tail:], "\n")
}

// tqdmProgressRe matches tqdm progress bars with current/total counts,
// e.g. " 45%|████▍ | 62/139 [...]". Captures: percentage, current, total.
// NOTE: Go regexp does not support negative lookahead; "Inferencing:" inner
// bars are filtered out in code (see GetTaskProgress).
var tqdmProgressRe = regexp.MustCompile(`(\d{1,3})%\|[^|]*\|\s*(\d+)/(\d+)`)

// GetTaskProgress parses the latest tqdm progress from log files for a running task.
// Returns the percentage (0-100), a human-readable progress text like "62/139",
// and a runningPhase ("infer" / "eval" / "") indicating which sub-phase is most active.
// Returns (-1, "", "") if no progress info is available.
func (s *EvalService) GetTaskProgress(ctx context.Context, evalTaskID string) (int, string, string) {
	task, err := s.taskRepo.GetByID(ctx, evalTaskID)
	if err != nil || task == nil {
		return -1, "", ""
	}

	// 即使非 running 也尝试推断 phase，便于前端在 parsing/succeeded 时也能展示对的高亮节点。
	phase := s.detectRunningPhase(ctx, evalTaskID)

	if task.Status != domain.EvalTaskStatusRunning {
		return -1, "", phase
	}

	candidates := s.candidateLogPaths(ctx, evalTaskID)
	best := pickFreshestExisting(candidates)
	if best == "" {
		return -1, "", phase
	}

	// Read last 16KB to find the latest outer tqdm line
	data, err := readTail(best, 16384)
	if err != nil || len(data) == 0 {
		return -1, "", phase
	}

	// Scan lines from the end; skip inner "Inferencing:" bars to find the
	// outer (main) progress bar first.
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Contains(line, "Inferencing:") {
			continue
		}
		m := tqdmProgressRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pct, err := strconv.Atoi(m[1])
		if err != nil || pct < 0 || pct > 100 {
			continue
		}
		return pct, m[2] + "/" + m[3], phase
	}
	return -1, "", phase
}

// detectRunningPhase 通过比较 logs/infer 和 logs/eval 下 .out 的最新 mtime 判断当前正在跑哪一阶段。
// 返回 "infer" / "eval" / ""。
func (s *EvalService) detectRunningPhase(ctx context.Context, evalTaskID string) string {
	root := s.config.OutputDir
	taskRoot := ""
	if root != "" {
		taskRoot = filepath.Join(root, evalTaskID)
	}
	if taskRoot == "" {
		if task, err := s.taskRepo.GetByID(ctx, evalTaskID); err == nil && task != nil && task.OutputDir != "" {
			taskRoot = task.OutputDir
		}
	}
	if taskRoot == "" {
		return ""
	}

	latest := func(pattern string) time.Time {
		matches, _ := filepath.Glob(pattern)
		var best time.Time
		for _, p := range matches {
			st, err := os.Stat(p)
			if err != nil || st.IsDir() {
				continue
			}
			if st.ModTime().After(best) {
				best = st.ModTime()
			}
		}
		return best
	}

	inferAt := latest(filepath.Join(taskRoot, "*", "logs", "infer", "*", "*.out"))
	evalAt := latest(filepath.Join(taskRoot, "*", "logs", "eval", "*", "*.out"))

	if evalAt.IsZero() && inferAt.IsZero() {
		return ""
	}
	if evalAt.After(inferAt) {
		return "eval"
	}
	return "infer"
}

// readTail reads the last n bytes of a file.
func readTail(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 {
		return nil, nil
	}
	readN := int64(n)
	if readN > size {
		readN = size
	}
	buf := make([]byte, readN)
	_, err = f.ReadAt(buf, size-readN)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *EvalService) executeTask(ctx context.Context, task domain.EvalTask, input CreateEvalTaskInput) {
	evalConfig := s.toCoreEvalConfig(task, input)

	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusValidating, "", "", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
		return
	}
	if err := s.coreClient.ValidateEvalConfig(ctx, evalConfig); err != nil {
		s.failTask(ctx, task.ID, "VALIDATE_FAILED", err)
		return
	}

	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusBuilding, "", "", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
		return
	}
	buildResp, err := s.coreClient.BuildEvalConfig(ctx, evalConfig)
	if err != nil {
		s.failTask(ctx, task.ID, "BUILD_FAILED", err)
		return
	}

	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusRunning, buildResp.GetOutputDir(), "", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
		return
	}
	executeResp, err := s.coreClient.ExecuteEval(ctx, task.ID, evalConfig, buildResp.GetConfigPath(), buildResp.GetOutputDir(), "")
	if err != nil {
		if s.consumeCancelled(task.ID) {
			log.Printf("任务被用户终止 task=%s: %v", task.ID, err)
			return
		}
		s.failTask(ctx, task.ID, "EXECUTE_FAILED", err)
		return
	}
	// ExecuteEval 没报 RPC 错，但 Core 可能返回 ok=false（被 cancel）已转化为 error 路径，这里不需重复处理。
	if s.consumeCancelled(task.ID) {
		log.Printf("任务被用户终止 task=%s（ExecuteEval 已退出）", task.ID)
		return
	}

	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusParsing, executeResp.GetOutputDir(), "", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
		return
	}
	result, err := s.coreClient.ParseEvalResult(ctx, task.ID, executeResp.GetOutputDir())
	if err != nil {
		s.failTask(ctx, task.ID, "PARSE_FAILED", err)
		return
	}

	if err := s.saveResult(ctx, task.ID, result); err != nil {
		s.failTask(ctx, task.ID, "SAVE_RESULT_FAILED", err)
		return
	}

	// Core 通过 metadata.valid_metric_count 报告是否产出了有效指标。
	// 当 OpenCompass 返回 exit 0 但实际推理失败（例如 401 / 超时）时，summary 全是 "-"，需把任务标 failed。
	if reason := failureReason(result); reason != "" {
		if updateErr := s.taskRepo.UpdateStatus(ctx, task.ID, domain.EvalTaskStatusFailed, executeResp.GetOutputDir(), "NO_VALID_METRIC", reason); updateErr != nil {
			log.Printf("更新任务状态失败: %v", updateErr)
		}
		return
	}

	if err := s.updateStatus(ctx, task.ID, domain.EvalTaskStatusSucceeded, executeResp.GetOutputDir(), "", ""); err != nil {
		log.Printf("更新任务状态失败: %v", err)
	}
}

func failureReason(result *evalv1.EvalResult) string {
	if result == nil {
		return ""
	}
	metadata := result.GetMetadata()
	if metadata == nil {
		if len(result.GetMetrics()) == 0 {
			return "OpenCompass 未产生有效指标"
		}
		return ""
	}
	if metadata["valid_metric_count"] == "0" {
		reason := metadata["failure_reason"]
		if reason == "" {
			reason = "OpenCompass 未产生有效指标"
		}
		return reason
	}
	if len(result.GetMetrics()) == 0 {
		if reason := metadata["failure_reason"]; reason != "" {
			return reason
		}
		return "OpenCompass 未产生有效指标"
	}
	return ""
}

func (s *EvalService) toCoreEvalConfig(task domain.EvalTask, input CreateEvalTaskInput) *evalv1.EvalConfig {
	datasetParams := map[string]string{}
	datasetPath := ""

	// 如果数据集来源为 huggingface 或 custom，尝试从数据库获取额外信息
	if s.datasetRepo != nil && input.DatasetName != "" {
		if dataset, err := s.datasetRepo.GetByCode(context.Background(), input.DatasetName); err == nil && dataset != nil {
			// 使用数据集的 ConfigPath 作为 path（Core 端会判断是否为 .py 配置文件）
			if dataset.ConfigPath != "" {
				datasetPath = dataset.ConfigPath
			}
			if dataset.LocalPath != "" && datasetPath == "" {
				datasetPath = dataset.LocalPath
			}
			if dataset.FileFormat != "" {
				datasetParams["file_format"] = dataset.FileFormat
			}

			// 传递 evaluator_type（方案 B：内置固定选项）
			// 当用户选择了非默认 evaluator 时，同时传递 raw_data_path 让 Core 重新生成配置
			evaluatorType := input.EvaluatorType
			if evaluatorType == "" {
				evaluatorType = "rouge"
			}
			datasetParams["evaluator_type"] = evaluatorType
			if evaluatorType != "rouge" && dataset.LocalPath != "" {
				// 非默认 evaluator 时，传递原始数据文件路径，让 Core 用新 evaluator 重新生成配置
				datasetParams["raw_data_path"] = dataset.LocalPath
			}
		}
	}

	// runtime 参数：优先使用前端传入的值，否则回退到服务端配置。
	timeoutSecs := s.config.DefaultTimeoutSecs
	if input.Runtime.TimeoutSeconds > 0 {
		timeoutSecs = input.Runtime.TimeoutSeconds
	}
	maxWorkers := int32(1)
	if input.Runtime.MaxWorkers > 0 {
		maxWorkers = int32(input.Runtime.MaxWorkers)
	}
	// keepRawOutputs 默认为 true；只有前端明确传入时才使用其值。
	keepRawOutputs := true
	if input.Runtime.HasValues {
		keepRawOutputs = input.Runtime.KeepRawOutputs
	}

	return &evalv1.EvalConfig{
		Model: &evalv1.ModelConfig{
			Type:      evalv1.ModelType_MODEL_TYPE_REMOTE_API,
			Provider:  input.Provider,
			ModelName: input.ModelName,
			BaseUrl:   input.BaseURL,
			ApiKey:    input.APIKey,
			Params:    input.Params,
		},
		Dataset: &evalv1.DatasetConfig{
			Type:   datasetTypeToProto(input.DatasetType),
			Name:   input.DatasetName,
			Path:   datasetPath,
			Params: datasetParams,
		},
		Runtime: &evalv1.RuntimeConfig{
			WorkDir:        s.config.OutputDir,
			TimeoutSeconds: int32(timeoutSecs),
			MaxWorkers:     maxWorkers,
			KeepRawOutputs: keepRawOutputs,
		},
		ExtraParams: map[string]string{
			"eval_task_id": task.ID,
		},
	}
}

type metricDTO struct {
	Name        string            `json:"name"`
	Value       float64           `json:"value"`
	DisplayName string            `json:"displayName"`
	Description string            `json:"description"`
	Extra       map[string]string `json:"extra,omitempty"`
}

type artifactDTO struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

func (s *EvalService) saveResult(ctx context.Context, evalTaskID string, result *evalv1.EvalResult) error {
	metricDTOs := make([]metricDTO, 0, len(result.GetMetrics()))
	for _, m := range result.GetMetrics() {
		metricDTOs = append(metricDTOs, metricDTO{
			Name:        m.GetName(),
			Value:       m.GetValue(),
			DisplayName: m.GetDisplayName(),
			Description: m.GetDescription(),
			Extra:       m.GetExtra(),
		})
	}
	metrics, err := json.Marshal(metricDTOs)
	if err != nil {
		return fmt.Errorf("序列化指标失败: %w", err)
	}

	artifactDTOs := make([]artifactDTO, 0, len(result.GetArtifacts()))
	for _, a := range result.GetArtifacts() {
		artifactDTOs = append(artifactDTOs, artifactDTO{
			Type:        artifactTypeToString(a.GetType()),
			Name:        a.GetName(),
			Path:        a.GetPath(),
			Description: a.GetDescription(),
		})
	}
	artifacts, err := json.Marshal(artifactDTOs)
	if err != nil {
		return fmt.Errorf("序列化产物失败: %w", err)
	}

	metadata, err := json.Marshal(result.GetMetadata())
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	return s.resultRepo.Save(ctx, domain.EvalResult{
		EvalTaskID:    evalTaskID,
		MetricsJSON:   string(metrics),
		ArtifactsJSON: string(artifacts),
		RawResultPath: result.GetRawResultPath(),
		ReportPath:    result.GetReportPath(),
		LogPath:       result.GetLogPath(),
		MetadataJSON:  string(metadata),
	})
}

func artifactTypeToString(t evalv1.ArtifactType) string {
	switch t {
	case evalv1.ArtifactType_ARTIFACT_TYPE_CONFIG:
		return "config"
	case evalv1.ArtifactType_ARTIFACT_TYPE_RAW_RESULT:
		return "raw_result"
	case evalv1.ArtifactType_ARTIFACT_TYPE_REPORT:
		return "report"
	case evalv1.ArtifactType_ARTIFACT_TYPE_LOG:
		return "log"
	case evalv1.ArtifactType_ARTIFACT_TYPE_OTHER:
		return "other"
	default:
		return "other"
	}
}

func (s *EvalService) updateStatus(ctx context.Context, id string, status domain.EvalTaskStatus, outputDir string, errorCode string, errorMessage string) error {
	return s.taskRepo.UpdateStatus(ctx, id, status, outputDir, errorCode, errorMessage)
}

func (s *EvalService) failTask(ctx context.Context, id string, code string, err error) {
	if updateErr := s.updateStatus(ctx, id, domain.EvalTaskStatusFailed, "", code, err.Error()); updateErr != nil {
		log.Printf("更新失败状态失败: %v", updateErr)
	}
}

func (s *EvalService) saveModelPreset(ctx context.Context, input CreateEvalTaskInput) error {
	if s.modelRepo == nil {
		return nil
	}
	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.ModelName
	}
	return s.modelRepo.Save(ctx, domain.Model{
		UserID:      input.UserID,
		Provider:    input.Provider,
		ModelName:   input.ModelName,
		DisplayName: displayName,
		Version:     input.Version,
		APIKey:      input.APIKey,
		BaseURL:     input.BaseURL,
		MaskedKey:   maskAPIKey(input.APIKey),
	})
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

func datasetTypeToProto(datasetType string) evalv1.DatasetType {
	switch datasetType {
	case "opencompass_standard":
		return evalv1.DatasetType_DATASET_TYPE_OPENCOMPASS_STANDARD
	case "custom":
		return evalv1.DatasetType_DATASET_TYPE_CUSTOM
	case "huggingface":
		return evalv1.DatasetType_DATASET_TYPE_HUGGINGFACE
	default:
		return evalv1.DatasetType_DATASET_TYPE_OPENCOMPASS_DEMO
	}
}

func newEvalTaskID() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ev-%d", time.Now().UnixMilli()%1_000_000_000_000)
	}
	return "ev-" + hex.EncodeToString(b)
}

const maxEvalTaskNameLen = 200

func normalizeEvalTaskName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxEvalTaskNameLen {
		return s
	}
	return string([]rune(s)[:maxEvalTaskNameLen])
}
