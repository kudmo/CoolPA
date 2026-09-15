package optimizer

// ScaleMode indicates the direction of a scaling operation.
type ScaleMode int

const (
	// ScaleUpMode means resources should be increased.
	ScaleUpMode ScaleMode = iota
	// ScaleDownMode means resources should be decreased.
	ScaleDownMode
)

// ReactionType specifies the type of scaling reaction to apply.
type ReactionType int

const (
	HPA ReactionType = iota
	VPA
)

// OptimizedServiceState describes the target resource state for a
// single service after optimization.
type OptimizedServiceState struct {
	// ServiceName is the name of the service.
	ServiceName string

	// Replicas is the desired number of replicas (used for HPA).
	Replicas int

	// AppCPU is the desired CPU for the application container,
	// in millicores.
	AppCPU float64

	// PodCPU is the desired CPU for the entire pod, in millicores.
	PodCPU float64

	// AppMemory is the desired memory for the application container,
	// in megabytes.
	AppMemory float64

	// PodMemory is the desired memory for the entire pod, in megabytes.
	PodMemory float64

	// Reaction indicates whether to scale horizontally or vertically.
	Reaction ReactionType
}

// OptimizedState contains the target states for all services that
// require scaling in a single optimization cycle.
type OptimizedState struct {
	Services []OptimizedServiceState
}
