package domain

import "time"

type EvalResult struct {
	EvalTaskID    string
	MetricsJSON   string
	ArtifactsJSON string
	RawResultPath string
	ReportPath    string
	LogPath       string
	MetadataJSON  string
}

type Model struct {
	ID          int64
	UserID      int64
	Provider    string
	ModelName   string
	DisplayName string
	Version     string
	APIKey      string
	BaseURL     string
	MaskedKey   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
