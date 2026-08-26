package optimizer

import "time"

type ReactionOptimizerConfig struct {
	CpuStep      float64
	MemoryStep   float64
	ReplicasStep int

	TargetCpuUtilization float64
	Lambda               float64

	TimeWindow time.Duration
}
