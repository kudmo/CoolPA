package optimizer

type ScaleMode int

const (
	ScaleUpMode ScaleMode = iota
	ScaleDownMode
)

type ReactionType int

const (
	HPA ReactionType = iota
	VPA
)

type OptimizedServiceState struct {
	ServiceName string
	Replicas    int
	AppCPU      float64 // in mCore
	PodCPU      float64 // in mCore
	AppMemory   float64 // in MB
	PodMemory   float64 // in MB
	Reaction    ReactionType
}

type OptimizedState struct {
	Services []OptimizedServiceState
}
