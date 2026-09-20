package optimizer

import "context"

type Optimizer interface {
	RunOptimization(ctx context.Context, scalingNamespace string, services []string, mode ScaleMode) (OptimizedState, error)
}
