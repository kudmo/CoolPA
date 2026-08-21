package metrics

type ServiceInfo struct {
	Name         string            `json:"name"`
	InboundCalls []string          `json:"inbound_calls"`
	OuboundCalls []string          `json:"outbond_calls"`
	Labels       map[string]string `json:"labels,omitempty"`
	Description  string            `json:"description,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}
