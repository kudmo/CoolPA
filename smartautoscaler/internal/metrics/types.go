package metrics

// ServiceInfo contains descriptive information about a service
// that is relevant for autoscaling decisions. It includes the
// service's name and the names of services it communicates with
// (inbound and outbound calls), which can be used to build a
// service dependency graph.
type ServiceInfo struct {
	// Name is the unique identifier of the service within the
	// monitored namespace or cluster.
	Name string `json:"name"`

	// InboundCalls lists the names of services that call this service.
	// These are incoming dependencies.
	InboundCalls []string `json:"inbound_calls"`

	// OutboundCalls lists the names of services that this service calls.
	// These are outgoing dependencies.
	OutboundCalls []string `json:"outbond_calls"`
}
