package application

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"eval-dominator/backend/internal/config"
	"eval-dominator/backend/internal/domain"
)

type DatasetRepository interface {
	List(ctx context.Context, includeDisabled bool) ([]domain.Dataset, error)
	GetByID(ctx context.Context, id int64) (*domain.Dataset, error)
	GetByCode(ctx context.Context, code string) (*domain.Dataset, error)
	Create(ctx context.Context, dataset domain.Dataset) (*domain.Dataset, error)
	Update(ctx context.Context, id int64, dataset domain.Dataset) (*domain.Dataset, error)
	UpdateEnabled(ctx context.Context, id int64, enabled bool) error
	Delete(ctx context.Context, id int64) error
	UpsertBuiltin(ctx context.Context, dataset domain.Dataset) error
	UpsertHuggingFace(ctx context.Context, dataset domain.Dataset) error
	ListHuggingFaceRepos(ctx context.Context) ([]string, error)
}

// DatasetCoreClient 定义数据集操作所需的 Core gRPC 能力。
type DatasetCoreClient interface {
	PullHuggingFaceDataset(ctx context.Context, repo, subset, split, cacheDir string) (localPath string, sampleCount int, err error)
	PrepareCustomDataset(ctx context.Context, localPath, taskType string) (configPath string, sampleCount int, err error)
}

type DatasetService struct {
	repo       DatasetRepository
	coreClient DatasetCoreClient
	config     config.DatasetConfig
}

func NewDatasetService(repo DatasetRepository, cfg config.DatasetConfig) *DatasetService {
	return &DatasetService{repo: repo, config: cfg}
}

// SetCoreClient 设置 Core gRPC 客户端（在 main.go 中初始化后调用）。
func (s *DatasetService) SetCoreClient(client DatasetCoreClient) {
	s.coreClient = client
}

type CreateDatasetInput struct {
	Code          string
	DisplayName   string
	Description   string
	Type          string
	SampleCount   int
	InferenceMode string
	ConfigPath    string
	ExtraJSON     string
}

type UpdateDatasetInput struct {
	DisplayName   string
	Description   string
	Type          string
	SampleCount   int
	Enabled       bool
	InferenceMode string
	ConfigPath    string
	ExtraJSON     string
}

func (s *DatasetService) List(ctx context.Context, includeDisabled bool) ([]domain.Dataset, error) {
	return s.repo.List(ctx, includeDisabled)
}

func (s *DatasetService) GetByCode(ctx context.Context, code string) (*domain.Dataset, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *DatasetService) Create(ctx context.Context, input CreateDatasetInput) (*domain.Dataset, error) {
	if input.Code == "" {
		return nil, fmt.Errorf("数据集 code 不能为空")
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Code
	}
	if input.Type == "" {
		input.Type = "custom"
	}
	return s.repo.Create(ctx, domain.Dataset{
		Code:          input.Code,
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		Type:          input.Type,
		Source:        domain.DatasetSourceCustom,
		SampleCount:   input.SampleCount,
		Enabled:       true,
		InferenceMode: input.InferenceMode,
		ConfigPath:    input.ConfigPath,
		ExtraJSON:     input.ExtraJSON,
	})
}

func (s *DatasetService) Update(ctx context.Context, id int64, input UpdateDatasetInput) (*domain.Dataset, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := domain.Dataset{
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		Type:          input.Type,
		SampleCount:   input.SampleCount,
		Enabled:       input.Enabled,
		InferenceMode: input.InferenceMode,
		ConfigPath:    input.ConfigPath,
		ExtraJSON:     input.ExtraJSON,
	}
	if updated.DisplayName == "" {
		updated.DisplayName = existing.DisplayName
	}
	if updated.Type == "" {
		updated.Type = existing.Type
	}
	if updated.ConfigPath == "" {
		updated.ConfigPath = existing.ConfigPath
	}
	if updated.InferenceMode == "" {
		updated.InferenceMode = existing.InferenceMode
	}
	return s.repo.Update(ctx, id, updated)
}

func (s *DatasetService) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.UpdateEnabled(ctx, id, enabled)
}

func (s *DatasetService) Delete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.Source == domain.DatasetSourceBuiltin {
		return fmt.Errorf("内置数据集不可删除，可改为「禁用」")
	}
	return s.repo.Delete(ctx, id)
}

// SyncResult 用于 handler 给前端返回同步统计。
type SyncResult struct {
	Scanned  int      `json:"scanned"`
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// Sync 扫描 OpenCompass demo 目录，把每个 demo_*.py 入库（已存在的只更新 config_path）。
func (s *DatasetService) Sync(ctx context.Context) (*SyncResult, error) {
	dir, err := s.resolveDemoDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取 demo 目录失败: %w", err)
	}

	result := &SyncResult{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".py") || entry.Name() == "__init__.py" {
			continue
		}
		result.Scanned++

		full := filepath.Join(dir, entry.Name())
		code := strings.TrimSuffix(entry.Name(), ".py")
		meta := s.parseDemoMeta(code, full)

		existing, _ := s.repo.GetByCode(ctx, code)
		if existing == nil {
			result.Inserted++
		} else {
			result.Updated++
		}

		if err := s.repo.UpsertBuiltin(ctx, domain.Dataset{
			Code:          code,
			DisplayName:   meta.DisplayName,
			Description:   meta.Description,
			Type:          meta.Type,
			SampleCount:   meta.SampleCount,
			InferenceMode: meta.InferenceMode,
			ConfigPath:    full,
			ExtraJSON:     "{}",
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", code, err.Error()))
		}
	}
	return result, nil
}

// SyncOnStartup 在后端启动时调用，失败仅打日志，不阻塞启动。
func (s *DatasetService) SyncOnStartup(ctx context.Context) {
	result, err := s.Sync(ctx)
	if err != nil {
		log.Printf("数据集同步跳过: %v", err)
		return
	}
	log.Printf("数据集同步完成: scanned=%d, inserted=%d, updated=%d", result.Scanned, result.Inserted, result.Updated)
}

func (s *DatasetService) resolveDemoDir() (string, error) {
	if s.config.OpenCompassDemoDir != "" {
		path, err := filepath.Abs(s.config.OpenCompassDemoDir)
		if err == nil {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				return path, nil
			}
		}
	}

	// 默认尝试 ../core/.venv/lib/python3.*/site-packages/opencompass/configs/datasets/demo
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("无法定位工作目录: %w", err)
	}

	candidates := []string{
		filepath.Join(cwd, "..", "core"),
		cwd,
		filepath.Dir(cwd),
		filepath.Dir(filepath.Dir(cwd)),
	}
	for _, base := range candidates {
		matches, _ := filepath.Glob(filepath.Join(base, ".venv", "lib", "python3.*", "site-packages", "opencompass", "configs", "datasets", "demo"))
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && info.IsDir() {
				return m, nil
			}
		}
		matches, _ = filepath.Glob(filepath.Join(base, "core", ".venv", "lib", "python3.*", "site-packages", "opencompass", "configs", "datasets", "demo"))
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && info.IsDir() {
				return m, nil
			}
		}
	}
	return "", fmt.Errorf("未找到 OpenCompass demo 目录，可在配置文件 dataset.opencompass_demo_dir 显式指定")
}

type demoMeta struct {
	DisplayName   string
	Description   string
	Type          string
	SampleCount   int
	InferenceMode string // gen / ppl
}

var (
	demoBaseRe      = regexp.MustCompile(`^demo_([a-zA-Z0-9]+)_(chat|base)_(gen|ppl)$`)
	demoTestRangeRe = regexp.MustCompile(`test_range['"]\s*\]\s*=\s*['"]\[(\d+):(\d+)\]['"]`)
)

func (s *DatasetService) parseDemoMeta(code, fullPath string) demoMeta {
	meta := demoMeta{
		DisplayName: code,
		Description: "OpenCompass 内置 demo 数据集",
		Type:        "opencompass_demo",
	}

	if match := demoBaseRe.FindStringSubmatch(code); match != nil {
		base, mode, infer := match[1], match[2], match[3]
		meta.DisplayName = fmt.Sprintf("%s (%s · %s) · Demo", strings.ToUpper(base), titleCase(mode), strings.ToUpper(infer))
		meta.Description = fmt.Sprintf("OpenCompass 自带 %s demo（%s 模式 / %s 推理），用于快速验证评测链路。", strings.ToUpper(base), mode, infer)
		meta.InferenceMode = infer
	}

	if data, err := os.ReadFile(fullPath); err == nil {
		if m := demoTestRangeRe.FindStringSubmatch(string(data)); m != nil {
			start, _ := strconv.Atoi(m[1])
			end, _ := strconv.Atoi(m[2])
			if end > start {
				meta.SampleCount = end - start
			}
		}
	}

	return meta
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ------------------------------------------------------------------
// HuggingFace 搜索与拉取
// ------------------------------------------------------------------

// HuggingFaceSearchResult 表示 HuggingFace API 返回的单个搜索结果。
type HuggingFaceSearchResult struct {
	ID           string   `json:"id"`
	Author       string   `json:"author,omitempty"`
	Description  string   `json:"description,omitempty"`
	Downloads    int      `json:"downloads,omitempty"`
	Likes        int      `json:"likes,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	LastModified string   `json:"lastModified,omitempty"`
	Pulled       bool     `json:"pulled"` // 是否已拉取到本地
}

// HuggingFaceSearchParams 搜索参数。
type HuggingFaceSearchParams struct {
	Keyword string
	Sort    string // trending, downloads, likes, lastModified
	Tag     string // e.g. "task_categories:text-classification"
	Limit   int
}

// SearchHuggingFace 调用 HuggingFace REST API 搜索数据集。
// keyword 为空时返回热门数据集。
func (s *DatasetService) SearchHuggingFace(ctx context.Context, params HuggingFaceSearchParams) ([]HuggingFaceSearchResult, error) {
	limit := params.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	baseURL := "https://huggingface.co/api/datasets"
	if s.config.HuggingFaceMirror != "" {
		baseURL = strings.TrimSuffix(s.config.HuggingFaceMirror, "/") + "/api/datasets"
	}

	hfParams := url.Values{}
	hfParams.Set("limit", strconv.Itoa(limit))
	hfParams.Set("full", "true")

	if params.Keyword != "" {
		hfParams.Set("search", params.Keyword)
	}

	// 排序
	sort := params.Sort
	if sort == "" {
		sort = "trending"
	}
	if sort != "trending" {
		hfParams.Set("sort", sort)
		hfParams.Set("direction", "-1")
	}

	// 标签筛选
	if params.Tag != "" {
		hfParams.Set("filter", params.Tag)
	}

	reqURL := baseURL + "?" + hfParams.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * 1e9} // 30s
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 HuggingFace API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HuggingFace API 返回 %d: %s", resp.StatusCode, string(body))
	}

	var results []HuggingFaceSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("解析 HuggingFace API 响应失败: %w", err)
	}

	// 标记已拉取的数据集
	pulledRepos, _ := s.repo.ListHuggingFaceRepos(ctx)
	pulledSet := make(map[string]bool, len(pulledRepos))
	for _, r := range pulledRepos {
		pulledSet[r] = true
	}
	for i := range results {
		results[i].Pulled = pulledSet[results[i].ID]
	}

	return results, nil
}

// ------------------------------------------------------------------
// HuggingFace 数据集详情
// ------------------------------------------------------------------

// HFDatasetDetail HuggingFace 数据集详细信息。
type HFDatasetDetail struct {
	ID           string   `json:"id"`
	Author       string   `json:"author,omitempty"`
	Description  string   `json:"description,omitempty"`
	Downloads    int      `json:"downloads"`
	Likes        int      `json:"likes"`
	LastModified string   `json:"lastModified,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	License      string   `json:"license,omitempty"`
	Languages    []string `json:"languages,omitempty"`
	TaskTypes    []string `json:"taskTypes,omitempty"`
	FileFormats  []string `json:"fileFormats,omitempty"`
	TotalSize    int64    `json:"totalSize,omitempty"`
	FileCount    int      `json:"fileCount,omitempty"`
}

// GetHuggingFaceDetail 获取单个 HuggingFace 数据集的详细信息。
func (s *DatasetService) GetHuggingFaceDetail(ctx context.Context, repoID string) (*HFDatasetDetail, error) {
	if repoID == "" {
		return nil, fmt.Errorf("数据集 ID 不能为空")
	}

	baseURL := "https://huggingface.co/api/datasets"
	if s.config.HuggingFaceMirror != "" {
		baseURL = strings.TrimSuffix(s.config.HuggingFaceMirror, "/") + "/api/datasets"
	}

	reqURL := baseURL + "/" + repoID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * 1e9}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 HuggingFace API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HuggingFace API 返回 %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		ID           string          `json:"id"`
		Author       string          `json:"author"`
		Description  string          `json:"description"`
		Downloads    int             `json:"downloads"`
		Likes        int             `json:"likes"`
		LastModified string          `json:"lastModified"`
		Tags         []string        `json:"tags"`
		UsedStorage  int64           `json:"usedStorage"`
		CardData     json.RawMessage `json:"cardData"`
		Siblings     []struct {
			RFilename string `json:"rfilename"`
			Size      int64  `json:"size"`
		} `json:"siblings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	detail := &HFDatasetDetail{
		ID:           raw.ID,
		Author:       raw.Author,
		Description:  raw.Description,
		Downloads:    raw.Downloads,
		Likes:        raw.Likes,
		LastModified: raw.LastModified,
		Tags:         raw.Tags,
		FileCount:    len(raw.Siblings),
	}

	// 从 tags 解析分类信息
	for _, tag := range raw.Tags {
		switch {
		case strings.HasPrefix(tag, "task_categories:"):
			detail.TaskTypes = append(detail.TaskTypes, strings.TrimPrefix(tag, "task_categories:"))
		case strings.HasPrefix(tag, "language:"):
			detail.Languages = append(detail.Languages, strings.TrimPrefix(tag, "language:"))
		case strings.HasPrefix(tag, "license:"):
			if detail.License == "" {
				detail.License = strings.TrimPrefix(tag, "license:")
			}
		}
	}

	// 从 cardData 补充
	if raw.CardData != nil {
		var card map[string]interface{}
		if json.Unmarshal(raw.CardData, &card) == nil {
			if v, ok := card["license"]; ok {
				if s, ok := v.(string); ok && detail.License == "" {
					detail.License = s
				}
			}
			if v, ok := card["task_categories"]; ok {
				if arr, ok := v.([]interface{}); ok && len(detail.TaskTypes) == 0 {
					for _, t := range arr {
						if s, ok := t.(string); ok {
							detail.TaskTypes = append(detail.TaskTypes, s)
						}
					}
				}
			}
		}
	}

	// 从 siblings 推断文件格式
	formatSet := make(map[string]bool)
	var totalSize int64
	for _, sib := range raw.Siblings {
		ext := strings.ToLower(filepath.Ext(sib.RFilename))
		switch ext {
		case ".parquet", ".csv", ".jsonl", ".json", ".tsv", ".arrow", ".txt":
			formatSet[ext[1:]] = true // strip leading dot
		}
		if sib.Size > 0 {
			totalSize += sib.Size
		}
	}
	for f := range formatSet {
		detail.FileFormats = append(detail.FileFormats, f)
	}
	if totalSize > 0 {
		detail.TotalSize = totalSize
	} else if raw.UsedStorage > 0 {
		detail.TotalSize = raw.UsedStorage
	}

	return detail, nil
}

// PullHuggingFaceInput 拉取 HuggingFace 数据集的输入参数。
type PullHuggingFaceInput struct {
	Repo   string
	Subset string
	Split  string
}

// PullHuggingFace 通过 Core gRPC 下载 HuggingFace 数据集到本地，并入库。
func (s *DatasetService) PullHuggingFace(ctx context.Context, input PullHuggingFaceInput) (*domain.Dataset, error) {
	if input.Repo == "" {
		return nil, fmt.Errorf("HuggingFace 仓库名不能为空")
	}
	if s.coreClient == nil {
		return nil, fmt.Errorf("Core 服务未连接，无法拉取数据集")
	}

	cacheDir := s.config.DatasetStorageDir
	if cacheDir == "" {
		cacheDir = "../runtime/datasets"
	}

	localPath, sampleCount, err := s.coreClient.PullHuggingFaceDataset(ctx, input.Repo, input.Subset, input.Split, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("拉取 HuggingFace 数据集失败: %w", err)
	}

	// 生成数据集 code（用仓库名 + 子集）
	code := strings.ReplaceAll(input.Repo, "/", "__")
	if input.Subset != "" {
		code = code + "__" + input.Subset
	}

	// 通过 Core 生成 OpenCompass 可用的 .py 配置文件（PrepareCustomDataset 会校验格式并生成）。
	configPath, _, err := s.coreClient.PrepareCustomDataset(ctx, localPath, "qa")
	if err != nil {
		log.Printf("HF 数据集 PrepareCustomDataset 失败（repo=%s）: %v，将使用原始数据路径作为 ConfigPath", input.Repo, err)
		// 回退：ConfigPath 指向 .jsonl 数据文件，Core 端 _write_mmengine_config 有兜底逻辑
		configPath = localPath
	}

	ds := domain.Dataset{
		Code:              code,
		DisplayName:       input.Repo,
		Description:       fmt.Sprintf("HuggingFace 数据集 %s", input.Repo),
		Type:              "huggingface",
		Source:            domain.DatasetSourceHuggingFace,
		SampleCount:       sampleCount,
		Enabled:           true,
		InferenceMode:     "gen",
		ConfigPath:        configPath,
		ExtraJSON:         "{}",
		HuggingFaceRepo:   input.Repo,
		HuggingFaceSubset: input.Subset,
		LocalPath:         localPath,
		FileFormat:        "jsonl",
	}

	if err := s.repo.UpsertHuggingFace(ctx, ds); err != nil {
		return nil, fmt.Errorf("入库 HuggingFace 数据集失败: %w", err)
	}

	return s.repo.GetByCode(ctx, code)
}

// ------------------------------------------------------------------
// 自定义数据集（文件上传 / 本地路径）
// ------------------------------------------------------------------

// CreateCustomFromFileInput 上传文件创建自定义数据集的输入。
type CreateCustomFromFileInput struct {
	DisplayName string
	Description string
	TaskType    string // choice / qa / classification
	FileName    string
	FileData    io.Reader
}

// CreateCustomFromFile 接收上传文件，保存并通过 Core 验证格式后入库。
func (s *DatasetService) CreateCustomFromFile(ctx context.Context, input CreateCustomFromFileInput) (*domain.Dataset, error) {
	if input.FileName == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}
	if s.coreClient == nil {
		return nil, fmt.Errorf("Core 服务未连接，无法验证数据集")
	}

	storageDir := s.config.DatasetStorageDir
	if storageDir == "" {
		storageDir = "../runtime/datasets"
	}
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	// 生成唯一文件名
	code := "custom_" + strings.TrimSuffix(filepath.Base(input.FileName), filepath.Ext(input.FileName))
	destPath := filepath.Join(storageDir, code+filepath.Ext(input.FileName))

	// 写入文件
	f, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, input.FileData); err != nil {
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}
	f.Close()

	// 通过 Core 验证并生成配置
	configPath, sampleCount, err := s.coreClient.PrepareCustomDataset(ctx, destPath, input.TaskType)
	if err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("数据集验证失败: %w", err)
	}

	displayName := input.DisplayName
	if displayName == "" {
		displayName = code
	}

	ds := domain.Dataset{
		Code:          code,
		DisplayName:   displayName,
		Description:   input.Description,
		Type:          "custom",
		Source:        domain.DatasetSourceCustom,
		SampleCount:   sampleCount,
		Enabled:       true,
		InferenceMode: "gen",
		ConfigPath:    configPath,
		ExtraJSON:     "{}",
		LocalPath:     destPath,
		FileFormat:    strings.TrimPrefix(filepath.Ext(input.FileName), "."),
	}

	created, err := s.repo.Create(ctx, ds)
	if err != nil {
		return nil, fmt.Errorf("入库自定义数据集失败: %w", err)
	}
	return created, nil
}

// CreateCustomFromPathInput 通过本地路径创建自定义数据集的输入。
type CreateCustomFromPathInput struct {
	Code        string
	DisplayName string
	Description string
	LocalPath   string
	TaskType    string // choice / qa / classification
}

// CreateCustomFromPath 接收本地路径，通过 Core 验证格式后入库。
func (s *DatasetService) CreateCustomFromPath(ctx context.Context, input CreateCustomFromPathInput) (*domain.Dataset, error) {
	if input.LocalPath == "" {
		return nil, fmt.Errorf("本地路径不能为空")
	}
	if s.coreClient == nil {
		return nil, fmt.Errorf("Core 服务未连接，无法验证数据集")
	}

	// 验证文件存在
	if _, err := os.Stat(input.LocalPath); err != nil {
		return nil, fmt.Errorf("文件不存在: %w", err)
	}

	configPath, sampleCount, err := s.coreClient.PrepareCustomDataset(ctx, input.LocalPath, input.TaskType)
	if err != nil {
		return nil, fmt.Errorf("数据集验证失败: %w", err)
	}

	code := input.Code
	if code == "" {
		code = "custom_" + strings.TrimSuffix(filepath.Base(input.LocalPath), filepath.Ext(input.LocalPath))
	}
	displayName := input.DisplayName
	if displayName == "" {
		displayName = code
	}

	ds := domain.Dataset{
		Code:          code,
		DisplayName:   displayName,
		Description:   input.Description,
		Type:          "custom",
		Source:        domain.DatasetSourceCustom,
		SampleCount:   sampleCount,
		Enabled:       true,
		InferenceMode: "gen",
		ConfigPath:    configPath,
		ExtraJSON:     "{}",
		LocalPath:     input.LocalPath,
		FileFormat:    strings.TrimPrefix(filepath.Ext(input.LocalPath), "."),
	}

	created, err := s.repo.Create(ctx, ds)
	if err != nil {
		return nil, fmt.Errorf("入库自定义数据集失败: %w", err)
	}
	return created, nil
}

// ------------------------------------------------------------------
// Demo 数据集
// ------------------------------------------------------------------

// DemoDatasetInfo 表示内置 demo 数据集的信息。
type DemoDatasetInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	TaskType    string `json:"taskType"`
	FileFormat  string `json:"fileFormat"`
	SampleCount int    `json:"sampleCount"`
	Description string `json:"description"`
}

// ListDemoDatasets 返回内置 demo 数据集列表。
func (s *DatasetService) ListDemoDatasets() []DemoDatasetInfo {
	// 尝试从 Core 所在目录查找 demos
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "..", "core", "src", "opencompass_core", "demos"),
		filepath.Join(cwd, "core", "src", "opencompass_core", "demos"),
	}

	var demosDir string
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			demosDir = c
			break
		}
	}

	if demosDir == "" {
		return []DemoDatasetInfo{}
	}

	entries, err := os.ReadDir(demosDir)
	if err != nil {
		return []DemoDatasetInfo{}
	}

	var demos []DemoDatasetInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".csv" && ext != ".jsonl" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ext)
		taskType := "qa"
		if strings.Contains(name, "choice") {
			taskType = "choice"
		}
		demos = append(demos, DemoDatasetInfo{
			Name:        name,
			Path:        filepath.Join(demosDir, entry.Name()),
			TaskType:    taskType,
			FileFormat:  strings.TrimPrefix(ext, "."),
			Description: fmt.Sprintf("内置 demo 数据集 (%s)", taskType),
		})
	}
	return demos
}

// ------------------------------------------------------------------
// 数据集预览
// ------------------------------------------------------------------

// DatasetPreview 数据集预览结果。
type DatasetPreview struct {
	Headers      []string                 `json:"headers"`
	Rows         []map[string]interface{} `json:"rows"`
	Total        int                      `json:"total"`
	PreviewSize  int                      `json:"previewSize"`
	FileFormat   string                   `json:"fileFormat"`
	TotalColumns int                      `json:"totalColumns,omitempty"` // 实际总列数（当列被截断时返回）
}

// PreviewDataset 读取数据集文件的前 limit 行。
func (s *DatasetService) PreviewDataset(ctx context.Context, id int64, limit int) (*DatasetPreview, error) {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("数据集不存在: %w", err)
	}

	// 确定文件路径
	filePath := ds.LocalPath
	if filePath == "" {
		// 尝试从 ConfigPath 推断（对 demo 数据集等）
		filePath = s.resolveDataFileFromConfig(ds.ConfigPath)
	}
	if filePath == "" {
		return nil, fmt.Errorf("该数据集无本地文件，无法预览")
	}

	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	format := ds.FileFormat
	if format == "" {
		format = strings.TrimPrefix(filepath.Ext(filePath), ".")
	}

	switch strings.ToLower(format) {
	case "csv":
		return previewCSV(filePath, limit)
	case "jsonl":
		return previewJSONL(filePath, limit)
	case "json":
		return previewJSON(filePath, limit)
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", format)
	}
}

// resolveDataFileFromConfig 尝试从 .py 配置文件路径推断数据文件位置。
func (s *DatasetService) resolveDataFileFromConfig(configPath string) string {
	if configPath == "" {
		return ""
	}
	// 如果 configPath 本身就是数据文件
	ext := strings.ToLower(filepath.Ext(configPath))
	if ext == ".csv" || ext == ".jsonl" || ext == ".json" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}
	// 尝试同目录下找数据文件
	dir := filepath.Dir(configPath)
	for _, pattern := range []string{"*.csv", "*.jsonl", "*.json"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func previewCSV(filePath string, limit int) (*DatasetPreview, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true

	// 读 header
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("读取 CSV header 失败: %w", err)
	}

	var rows []map[string]interface{}
	total := 0
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		total++
		if len(rows) < limit {
			row := make(map[string]interface{})
			for i, h := range headers {
				if i < len(record) {
					row[h] = record[i]
				}
			}
			rows = append(rows, row)
		}
	}

	return &DatasetPreview{
		Headers:     headers,
		Rows:        rows,
		Total:       total,
		PreviewSize: len(rows),
		FileFormat:  "csv",
	}, nil
}

func previewJSONL(filePath string, limit int) (*DatasetPreview, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	headerSet := make(map[string]bool)
	var allRows []map[string]interface{}
	total := 0
	maxCols := 50 // 最多返回 50 列，避免前端表格卡死

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		total++
		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		if len(allRows) < limit {
			for k := range row {
				headerSet[k] = true
			}
			allRows = append(allRows, row)
		}
	}

	// 收集 Headers，保留插入顺序并截断
	var headers []string
	for k := range headerSet {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	totalCols := len(headers)
	if len(headers) > maxCols {
		headers = headers[:maxCols]
	}
	if len(headers) == 0 {
		headers = []string{"value"}
	}

	// 当列数远大于行数时（如 benchmark 数据集每列是一个 query），
	// 转置展示：每行一个 query，每列一个 sample。
	if totalCols > 10 && totalCols > len(allRows)*5 {
		numSamples := len(allRows)
		// 新 headers: query + sample_0, sample_1, ...
		newHeaders := make([]string, 0, numSamples+1)
		newHeaders = append(newHeaders, "query")
		for i := 0; i < numSamples; i++ {
			newHeaders = append(newHeaders, fmt.Sprintf("sample_%d", i))
		}
		// 转置：原列 key → 新行，原行 → 新列
		transposedRows := make([]map[string]interface{}, 0, len(headers))
		for _, key := range headers {
			newRow := map[string]interface{}{"query": key}
			for i, row := range allRows {
				if v, ok := row[key]; ok {
					newRow[fmt.Sprintf("sample_%d", i)] = v
				}
			}
			transposedRows = append(transposedRows, newRow)
		}
		headers = newHeaders
		allRows = transposedRows
		total = totalCols // 总行数变为原列数
		totalCols = numSamples + 1
	}

	// 截断超长单元格值
	for _, row := range allRows {
		for _, h := range headers {
			if v, ok := row[h]; ok {
				row[h] = truncatePreviewValue(v, 500)
			}
		}
	}

	result := &DatasetPreview{
		Headers:     headers,
		Rows:        allRows,
		Total:       total,
		PreviewSize: len(allRows),
		FileFormat:  "jsonl",
	}
	if totalCols > maxCols {
		result.TotalColumns = totalCols
	}
	return result, nil
}

// truncatePreviewValue 截断预览单元格值，避免超长内容导致前端卡死。
func truncatePreviewValue(v interface{}, maxLen int) interface{} {
	switch val := v.(type) {
	case string:
		if len(val) > maxLen {
			return val[:maxLen] + "..."
		}
	case []interface{}:
		b, _ := json.Marshal(val)
		s := string(b)
		if len(s) > maxLen {
			return s[:maxLen] + "..."
		}
		return s
	case map[string]interface{}:
		b, _ := json.Marshal(val)
		s := string(b)
		if len(s) > maxLen {
			return s[:maxLen] + "..."
		}
		return s
	}
	return v
}

func previewJSON(filePath string, limit int) (*DatasetPreview, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		// 尝试作为单个对象
		var single map[string]interface{}
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return nil, fmt.Errorf("JSON 解析失败: %w", err)
		}
		items = []map[string]interface{}{single}
	}

	headerSet := make(map[string]bool)
	var rows []map[string]interface{}
	for i, item := range items {
		if i >= limit {
			break
		}
		for k := range item {
			headerSet[k] = true
		}
		rows = append(rows, item)
	}

	var headers []string
	for k := range headerSet {
		headers = append(headers, k)
	}

	return &DatasetPreview{
		Headers:     headers,
		Rows:        rows,
		Total:       len(items),
		PreviewSize: len(rows),
		FileFormat:  "json",
	}, nil
}

// PreviewByPath 通过文件路径直接预览数据集。
func (s *DatasetService) PreviewByPath(ctx context.Context, filePath string, limit int) (*DatasetPreview, error) {
	if filePath == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	format := strings.TrimPrefix(filepath.Ext(filePath), ".")
	switch strings.ToLower(format) {
	case "csv":
		return previewCSV(filePath, limit)
	case "jsonl":
		return previewJSONL(filePath, limit)
	case "json":
		return previewJSON(filePath, limit)
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", format)
	}
}
