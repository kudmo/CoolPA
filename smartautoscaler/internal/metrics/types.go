package metrics

// ServiceID uniquely identifies a service within the cluster.
// The (Namespace, Name) pair is the service's unique ID.
type ServiceID struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

func (id *ServiceID) ToString() string {
	return id.Namespace + "_" + id.Name
}

// ServiceInfo contains descriptive information about a service
// that is relevant for autoscaling decisions. It includes the
// service's unique identifier (namespace + name) and the unique
// identifiers of the services it communicates with (inbound and
// outbound calls), which can be used to build a service
// dependency graph.
type ServiceInfo struct {
	// ID is the unique identifier of the service
	ID ServiceID `json:"name"`

	// InboundCalls lists the services that call this service.
	// These are incoming dependencies.
	InboundCalls []ServiceID `json:"inbound_calls"`

	// OutboundCalls lists the services that this service calls.
	// These are outgoing dependencies.
	OutboundCalls []ServiceID `json:"outbound_calls"`
}
