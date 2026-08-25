package metrics

type ServiceInfo struct {
	Name         string   `json:"name"`
	InboundCalls []string `json:"inbound_calls"`
	OuboundCalls []string `json:"outbond_calls"`
}
