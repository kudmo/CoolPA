// Package applier defines the interface for applying scaling actions.
//
// Implementations of the Applier interface are responsible for
// executing horizontal and vertical scaling operations.
package applier

import "context"

// Applier is the contract for applying scaling reactions to workloads.
//
// Implementations must be safe for concurrent use, as multiple
// scaling operations may be triggered simultaneously. Methods should
// return an error if the scaling operation cannot be performed,
// allowing callers to handle failures appropriately (e.g., retry,
// log, or alert).
//
// Implementations are expected to be idempotent where possible:
// applying the same scaling parameters multiple times should not
// produce unintended side effects beyond the desired state.
type Applier interface {
	// ApplyHPS scales the number of replicas for a given workload
	// (horizontal pod scaling).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts. Implementations
	//     should respect context deadlines and return an error if
	//     the operation is cancelled or times out.
	//   - namespace: namespace of the workload.
	//   - workload: Name of the workload
	//   - replicas: Desired number of replicas. Must be a non-negative
	//     integer.
	//
	// Returns:
	//   - nil if the scaling operation was successfully applied.
	//   - An error describing the failure reason if the operation
	//     could not be completed (e.g., workload not found,
	//     insufficient permissions, API unreachable).
	ApplyHPS(ctx context.Context, namespace string, workload string, replicas int32) error

	// ApplyVPS adjusts the resource requests/limits for a given workload
	// (vertical pod scaling).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts. Implementations
	//     should respect context deadlines and return an error if
	//     the operation is cancelled or times out.
	//   - namespace: namespace of the workload.
	//   - workload: Name of the workload
	//   - cpu: Desired CPU resource specification (e.g., "500m", "1").
	//     The exact format depends on the implementation but should
	//     follow Kubernetes resource quantity conventions.
	//   - memory: Desired memory resource specification (e.g., "128Mi",
	//     "1Gi"). Follows Kubernetes resource quantity conventions.
	//
	// Returns:
	//   - nil if the scaling operation was successfully applied.
	//   - An error describing the failure reason if the operation
	//     could not be completed.
	ApplyVPS(ctx context.Context, namespace string, workload string, cpu string, memory string) error
}
