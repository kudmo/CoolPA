package optimizer

import "context"

type Optimizer interface {
	RunOptimization(ctx context.Context, services []string, mode ScaleMode) (OptimizedState, error)
}
