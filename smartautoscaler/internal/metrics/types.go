package metrics

import "github.com/kudmo/CoolPA/internal/core"

// ServiceInfo contains descriptive information about a service
// that is relevant for autoscaling decisions. It includes the
// service's unique identifier (namespace + name) and the unique
// identifiers of the services it communicates with (inbound and
// outbound calls), which can be used to build a service
// dependency graph.
type ServiceInfo struct {
	// ID is the unique identifier of the service
	ID core.ServiceID `json:"name"`

	// InboundCalls lists the services that call this service.
	// These are incoming dependencies.
	InboundCalls []core.ServiceID `json:"inbound_calls"`

	// OutboundCalls lists the services that this service calls.
	// These are outgoing dependencies.
	OutboundCalls []core.ServiceID `json:"outbound_calls"`
}
