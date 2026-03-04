package constraints

import "github.com/kudmo/CoolPA/decision/ga/genome"

// ServicePolicy defines allowed reaction types and resource bounds for a single service.
type ServicePolicy struct {
	AllowedReactions []genome.ReactionType
	MinReplicas      int
	MaxReplicas      int
	MinCPU           float64
	MaxCPU           float64
}

// GlobalPolicy defines cluster-wide constraints.
type GlobalPolicy struct {
	ClusterCPULimit     float64
	ClusterReplicaLimit int
}

// ConstraintEngine holds per-service and global policies.
type ConstraintEngine struct {
	ServicePolicies map[string]ServicePolicy
	GlobalPolicy    GlobalPolicy
}

// GetAllowedReactions returns allowed reaction types for a service (or nil).
func (ce *ConstraintEngine) GetAllowedReactions(service string) []genome.ReactionType {
	if ce == nil {
		return nil
	}
	if sp, ok := ce.ServicePolicies[service]; ok {
		return sp.AllowedReactions
	}
	return nil
}
