package quotas

type ServiceQuotaID uint8

const (
	CpuQuota ServiceQuotaID = iota
	MemoryQuotasage
	PodsCount

	ServiceQuotaCount
)

type ServiceQuotas struct {
	Quotas [ServiceQuotaCount]int64
}

type NamespaceLimitID uint8

const (
	NamespaceMaxCpu NamespaceLimitID = iota
	NamespaceMaxMem
	NamespaceMaxPods

	NamespaceLimitCount
)

type ServiceLimitRangeID uint8

const (
	ServiceMinCpu ServiceLimitRangeID = iota
	ServiceMaxCpu

	ServiceMinMem
	ServiceMaxMem

	ServiceLimitRangeCount
)

type QuotasStorage struct {
	NamespaceLimits [NamespaceLimitCount]int64
	ServiceLimits   [ServiceLimitRangeCount]int64
	// ServiceQuotas   map[string]ServiceQuotas
}
