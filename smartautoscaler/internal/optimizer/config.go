package optimizer

import "time"

// ReactionOptimizerConfig holds parameters that control how the
// optimizer adjusts resources during scaling.
type ReactionOptimizerConfig struct {
	// The granularity of CPU adjustments. CPU changes
	// will be multiples of this value (e.g., 100m).
	CpuStep float64

	// The granularity of memory adjustments. Memory
	// changes will be multiples of this value (e.g., 256Mi).
	MemoryStep float64

	// The granularity of replica count adjustments.
	// Replica changes will be multiples of this value.
	ReplicasStep int

	// The desired CPU utilization (0..1).
	// It is not strictly enforced but used as a guide when setting
	// resource limits during optimization.
	TargetCpuUtilization float64

	// The value balances SLO compliance against resource savings.
	// Range: 0..1.
	//   0 — ignore SLO risk entirely (maximize savings)
	//   1 — prioritize SLO compliance (use more resources)
	Lambda float64

	// Time duration used to query metric ranges for analysis.
	TimeWindow time.Duration
}
