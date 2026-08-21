package applier

import "context"

type Applier interface {
	ApplyHPS(ctx context.Context, namespace string, workload string, replicas int32) error
	ApplyVPS(ctx context.Context, namespace string, workload string, cpu string, memory string) error
}
