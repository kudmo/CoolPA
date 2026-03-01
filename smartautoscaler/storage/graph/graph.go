package graph

import (
	"fmt"
	"sync"
	"time"

	"github.com/kudmo/CoolPA/storage/metrics"
)

type CallGraph struct {
	window time.Duration
	step   time.Duration
	nodes  map[string]*ServiceNode
	mu     sync.RWMutex
}

type ServiceNode struct {
	Name          string
	InboundEdges  map[string]*ServiceEdgeMetrics
	OutboundEdges map[string]*ServiceEdgeMetrics

	RequestCount  *metrics.RingWindow // istio_requests_total
	BytesSent     *metrics.RingWindow // istio_tcp_sent_bytes_total
	BytesReceived *metrics.RingWindow // istio_tcp_received_bytes_total
}

type ServiceEdgeMetrics struct {
	Latency95 *metrics.RingWindow // istio_request_duration_p95
	Latency50 *metrics.RingWindow // istio_request_duration_p50
}

// MetricID identifies types of metrics stored in the call graph.
type MetricID uint8

const (
	// Service-level metrics
	ServiceRequestCount MetricID = iota
	ServiceBytesSent
	ServiceBytesReceived

	// Edge-level metrics
	EdgeLatency95
	EdgeLatency50
)

// NewCallGraph creates a new call graph with given window and step for internal RingWindows.
func NewCallGraph(window, step time.Duration) *CallGraph {
	return &CallGraph{
		window: window,
		step:   step,
		nodes:  make(map[string]*ServiceNode),
	}
}

// GetServices returns the list of known services.
func (g *CallGraph) GetServices() []string {
	out := make([]string, 0, len(g.nodes))
	for name := range g.nodes {
		out = append(out, name)
	}
	return out
}

// getOrCreateNode returns existing node or creates a new one.
func (g *CallGraph) getOrCreateNode(name string) *ServiceNode {
	g.mu.Lock()
	defer g.mu.Unlock()

	n, ok := g.nodes[name]
	if ok {
		return n
	}

	n = &ServiceNode{
		Name:          name,
		InboundEdges:  make(map[string]*ServiceEdgeMetrics),
		OutboundEdges: make(map[string]*ServiceEdgeMetrics),
		RequestCount:  metrics.NewRingWindow(g.window, g.step),
		BytesSent:     metrics.NewRingWindow(g.window, g.step),
		BytesReceived: metrics.NewRingWindow(g.window, g.step),
	}

	g.nodes[name] = n
	return n
}

// GetNode returns a node by name. Caller should not mutate returned node maps concurrently.
func (g *CallGraph) GetNode(name string) (*ServiceNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[name]
	return n, ok
}

// ensureEdge makes sure edge metrics exist for given node and peer
func (g *CallGraph) ensureEdge(n *ServiceNode, peer string) *ServiceEdgeMetrics {
	em, ok := n.OutboundEdges[peer]
	if ok {
		return em
	}
	em = &ServiceEdgeMetrics{
		Latency95: metrics.NewRingWindow(g.window, g.step),
		Latency50: metrics.NewRingWindow(g.window, g.step),
	}
	n.OutboundEdges[peer] = em
	return em
}

// AddServiceMetric records a service-level metric identified by MetricID.
// For counter-like metrics use AddDelta semantics, for gauge-like metrics use Add.
func (g *CallGraph) AddServiceMetric(service string, ts time.Time, id MetricID, value float64) error {
	n := g.getOrCreateNode(service)
	switch id {
	case ServiceRequestCount:
		n.RequestCount.AddDelta(ts, value)
	case ServiceBytesSent:
		n.BytesSent.AddDelta(ts, value)
	case ServiceBytesReceived:
		n.BytesReceived.AddDelta(ts, value)
	default:
		return fmt.Errorf("invalid service metric id: %d", id)
	}
	return nil
}

// AddEdgeMetric records an edge-level metric (latency p95/p50) for a call from->to.
func (g *CallGraph) AddEdgeMetric(from, to string, ts time.Time, id MetricID, value float64) error {
	src := g.getOrCreateNode(from)
	dst := g.getOrCreateNode(to)

	switch id {
	case EdgeLatency95:
		out := g.ensureEdge(src, to)
		out.Latency95.Add(ts, value)

		in, ok := dst.InboundEdges[from]
		if !ok {
			in = &ServiceEdgeMetrics{
				Latency95: metrics.NewRingWindow(g.window, g.step),
				Latency50: metrics.NewRingWindow(g.window, g.step),
			}
			dst.InboundEdges[from] = in
		}
		in.Latency95.Add(ts, value)
	case EdgeLatency50:
		out := g.ensureEdge(src, to)
		out.Latency50.Add(ts, value)

		in, ok := dst.InboundEdges[from]
		if !ok {
			in = &ServiceEdgeMetrics{
				Latency95: metrics.NewRingWindow(g.window, g.step),
				Latency50: metrics.NewRingWindow(g.window, g.step),
			}
			dst.InboundEdges[from] = in
		}
		in.Latency50.Add(ts, value)
	default:
		return fmt.Errorf("invalid edge metric id: %d", id)
	}

	return nil
}

// RemoveService removes a service node and its edges.
func (g *CallGraph) RemoveService(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.nodes, name)
	// remove references to this service from other nodes
	for _, n := range g.nodes {
		delete(n.OutboundEdges, name)
		delete(n.InboundEdges, name)
	}
}

// SyncServices keeps only services present in active slice.
func (g *CallGraph) SyncServices(active []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	activeSet := make(map[string]struct{}, len(active))
	for _, s := range active {
		activeSet[s] = struct{}{}
	}

	for name := range g.nodes {
		if _, ok := activeSet[name]; !ok {
			delete(g.nodes, name)
		}
	}

	// clean edges to non-active services
	for _, n := range g.nodes {
		for peer := range n.OutboundEdges {
			if _, ok := activeSet[peer]; !ok {
				delete(n.OutboundEdges, peer)
			}
		}
		for peer := range n.InboundEdges {
			if _, ok := activeSet[peer]; !ok {
				delete(n.InboundEdges, peer)
			}
		}
	}
}
