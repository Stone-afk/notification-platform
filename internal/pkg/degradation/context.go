package degradation

import "context"

type degradationKey struct{}

func WithDegradation(ctx context.Context) context.Context {
	return context.WithValue(ctx, degradationKey{}, true)
}

func IsDegraded(ctx context.Context) bool {
	return ctx.Value(degradationKey{}) == true
}
