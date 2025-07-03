package sequential

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/service/provider"
)

var (
	_ provider.Selector        = (*selector)(nil)
	_ provider.SelectorBuilder = (*SelectorBuilder)(nil)
)

// selector 供应商顺序选择器
type selector struct {
	idx       int
	providers []provider.Provider
}

func (r *selector) Next(_ context.Context, _ domain.Notification) (provider.Provider, error) {
	if len(r.providers) == r.idx {
		return nil, fmt.Errorf("%w", errs.ErrNoAvailableProvider)
	}

	p := r.providers[r.idx]
	r.idx++
	return p, nil
}

type SelectorBuilder struct {
	providers []provider.Provider
}

func NewSelectorBuilder(providers []provider.Provider) *SelectorBuilder {
	return &SelectorBuilder{providers: providers}
}

func (s *SelectorBuilder) Build() (provider.Selector, error) {
	return &selector{
		providers: s.providers,
	}, nil
}
