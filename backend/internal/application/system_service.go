package application

import (
	"context"
)

type CoreHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type SystemService struct {
	core CoreHealthChecker
}

func NewSystemService(core CoreHealthChecker) *SystemService {
	return &SystemService{core: core}
}

type HealthStatus struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type HealthReport struct {
	Backend HealthStatus `json:"backend"`
	Core    HealthStatus `json:"core"`
}

func (s *SystemService) Health(ctx context.Context) HealthReport {
	report := HealthReport{
		Backend: HealthStatus{OK: true, Message: "ok"},
		Core:    HealthStatus{OK: true, Message: "ok"},
	}
	if s.core == nil {
		report.Core = HealthStatus{OK: false, Message: "core 未配置"}
		return report
	}
	if err := s.core.HealthCheck(ctx); err != nil {
		report.Core = HealthStatus{OK: false, Message: err.Error()}
	}
	return report
}
