package application

import (
	"context"
	"fmt"

	"eval-dominator/backend/internal/domain"
)

type ModelRepositoryFull interface {
	Create(ctx context.Context, model domain.Model) (*domain.Model, error)
	Update(ctx context.Context, id int64, userID int64, fields domain.Model, updateAPIKey bool) (*domain.Model, error)
	Delete(ctx context.Context, id int64, userID int64) error
	GetByID(ctx context.Context, id int64) (*domain.Model, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Model, error)
}

type ModelService struct {
	repo ModelRepositoryFull
}

func NewModelService(repo ModelRepositoryFull) *ModelService {
	return &ModelService{repo: repo}
}

type CreateModelInput struct {
	UserID      int64
	Provider    string
	ModelName   string
	DisplayName string
	Version     string
	BaseURL     string
	APIKey      string
}

type UpdateModelInput struct {
	Provider    string
	ModelName   string
	DisplayName string
	Version     string
	BaseURL     string
	APIKey      string
}

func (s *ModelService) List(ctx context.Context, userID int64) ([]domain.Model, error) {
	return s.repo.ListByUser(ctx, userID)
}

// minAPIKeyLength 是 API Key 的最小长度，避免用户误把模型名/baseUrl 填到 apiKey 字段。
// 主流厂商：OpenAI sk-... 51 字符、Anthropic 108 字符、阿里云 dashscope 35 字符；16 是非常宽松的下限。
const minAPIKeyLength = 16

func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	if len(apiKey) < minAPIKeyLength {
		return fmt.Errorf("API Key 长度过短（< %d），请确认是否填错（不要把模型名或 base_url 填到此处）", minAPIKeyLength)
	}
	return nil
}

func (s *ModelService) Create(ctx context.Context, input CreateModelInput) (*domain.Model, error) {
	if input.Provider == "" {
		return nil, fmt.Errorf("服务商不能为空")
	}
	if input.ModelName == "" {
		return nil, fmt.Errorf("模型名称不能为空")
	}
	if err := validateAPIKey(input.APIKey); err != nil {
		return nil, err
	}
	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.ModelName
	}
	return s.repo.Create(ctx, domain.Model{
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

// Update 更新模型预设：input 中为空的字段保留 existing 原值，避免单字段编辑误清空其他字段；
// apiKey 留空表示"不修改"（沿用旧值），填了则要通过最小长度校验。
func (s *ModelService) Update(ctx context.Context, id int64, userID int64, input UpdateModelInput) (*domain.Model, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.UserID != userID {
		return nil, fmt.Errorf("模型预设不存在")
	}

	updated := domain.Model{
		Provider:    fallbackString(input.Provider, existing.Provider),
		ModelName:   fallbackString(input.ModelName, existing.ModelName),
		DisplayName: fallbackString(input.DisplayName, existing.DisplayName),
		Version:     input.Version,
		BaseURL:     fallbackString(input.BaseURL, existing.BaseURL),
	}
	if updated.DisplayName == "" {
		updated.DisplayName = updated.ModelName
	}

	updateAPIKey := input.APIKey != ""
	if updateAPIKey {
		if err := validateAPIKey(input.APIKey); err != nil {
			return nil, err
		}
		updated.APIKey = input.APIKey
		updated.MaskedKey = maskAPIKey(input.APIKey)
	}
	return s.repo.Update(ctx, id, userID, updated, updateAPIKey)
}

func fallbackString(input, existing string) string {
	if input == "" {
		return existing
	}
	return input
}

func (s *ModelService) Delete(ctx context.Context, id int64, userID int64) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *ModelService) GetByID(ctx context.Context, id int64, userID int64) (*domain.Model, error) {
	model, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if model.UserID != userID {
		return nil, fmt.Errorf("模型预设不存在")
	}
	return model, nil
}
