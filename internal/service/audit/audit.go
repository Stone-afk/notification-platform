package audit

import (
	"context"
	"notification-platform/internal/domain"
)

type service struct{}

func NewService() Service {
	return &service{}
}

func (s *service) CreateAudit(_ context.Context, _ domain.Audit) (int64, error) {
	// TODO implement me
	panic("implement me")
}
