package application

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
}

type DatasetService struct {
	repo   DatasetRepository
	config config.DatasetConfig
}

func NewDatasetService(repo DatasetRepository, cfg config.DatasetConfig) *DatasetService {
	return &DatasetService{repo: repo, config: cfg}
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
	Scanned int      `json:"scanned"`
	Inserted int     `json:"inserted"`
	Updated  int     `json:"updated"`
	Skipped  int     `json:"skipped"`
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
	demoBaseRe       = regexp.MustCompile(`^demo_([a-zA-Z0-9]+)_(chat|base)_(gen|ppl)$`)
	demoTestRangeRe  = regexp.MustCompile(`test_range['"]\s*\]\s*=\s*['"]\[(\d+):(\d+)\]['"]`)
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
