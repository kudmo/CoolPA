package graph

import (
	"testing"
	"time"
)

func TestAddServiceSample(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	g := NewCallGraph(window, step)

	base := time.Now().Truncate(step)

	// use new AddServiceMetric API
	if err := g.AddServiceMetric("svc1", base, ServiceRequestCount, 5); err != nil {
		t.Fatalf("AddServiceMetric error: %v", err)
	}
	if err := g.AddServiceMetric("svc1", base, ServiceBytesSent, 1000); err != nil {
		t.Fatalf("AddServiceMetric error: %v", err)
	}
	if err := g.AddServiceMetric("svc1", base, ServiceBytesReceived, 2000); err != nil {
		t.Fatalf("AddServiceMetric error: %v", err)
	}

	n, ok := g.GetNode("svc1")
	if !ok {
		t.Fatal("expected node svc1 to exist")
	}

	if got := n.RequestCount.Sum(); got != 5 {
		t.Fatalf("RequestCount.Sum() = %v, want 5", got)
	}
	if got := n.BytesSent.Sum(); got != 1000 {
		t.Fatalf("BytesSent.Sum() = %v, want 1000", got)
	}
	if got := n.BytesReceived.Sum(); got != 2000 {
		t.Fatalf("BytesReceived.Sum() = %v, want 2000", got)
	}

	// adding another delta should accumulate for RequestCount and bytes
	if err := g.AddServiceMetric("svc1", base, ServiceRequestCount, 2); err != nil {
		t.Fatalf("AddServiceMetric error: %v", err)
	}
	if err := g.AddServiceMetric("svc1", base, ServiceBytesSent, 5); err != nil {
		t.Fatalf("AddServiceMetric error: %v", err)
	}
	if err := g.AddServiceMetric("svc1", base, ServiceBytesReceived, 10); err != nil {
		t.Fatalf("AddServiceMetric error: %v", err)
	}

	if got := n.RequestCount.Sum(); got != 7 {
		t.Fatalf("after second sample RequestCount.Sum() = %v, want 7", got)
	}
}

func TestAddEdgeSampleCreatesBothSides(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	g := NewCallGraph(window, step)

	base := time.Now().Truncate(step)

	if err := g.AddEdgeMetric("svc-a", "svc-b", base, EdgeLatency95, 50); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}
	if err := g.AddEdgeMetric("svc-a", "svc-b", base, EdgeLatency50, 20); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}

	src, ok := g.GetNode("svc-a")
	if !ok {
		t.Fatal("src node missing")
	}
	out, ok := src.OutboundEdges["svc-b"]
	if !ok {
		t.Fatal("expected outbound edge svc-a->svc-b")
	}
	if got := out.Latency95.Sum(); got != 50 {
		t.Fatalf("out.Latency95.Sum() = %v, want 50", got)
	}

	dst, ok := g.GetNode("svc-b")
	if !ok {
		t.Fatal("dst node missing")
	}
	in, ok := dst.InboundEdges["svc-a"]
	if !ok {
		t.Fatal("expected inbound edge svc-a->svc-b on dst")
	}
	if got := in.Latency50.Sum(); got != 20 {
		t.Fatalf("in.Latency50.Sum() = %v, want 20", got)
	}
}

func TestRemoveServiceAndEdgeCleanup(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	g := NewCallGraph(window, step)

	base := time.Now().Truncate(step)

	if err := g.AddEdgeMetric("a", "b", base, EdgeLatency95, 1); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}
	if err := g.AddEdgeMetric("a", "b", base, EdgeLatency50, 1); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}
	if err := g.AddEdgeMetric("a", "b", base, EdgeLatency95, 2); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}
	if err := g.AddEdgeMetric("c", "b", base, EdgeLatency50, 2); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}

	// ensure nodes present
	if _, ok := g.GetNode("b"); !ok {
		t.Fatal("b should exist before removal")
	}

	g.RemoveService("b")

	if _, ok := g.GetNode("b"); ok {
		t.Fatal("b should be removed")
	}

	// ensure other nodes don't keep edges to b
	a, _ := g.GetNode("a")
	if _, ok := a.OutboundEdges["b"]; ok {
		t.Fatal("outbound edge to removed service should be deleted")
	}
	c, _ := g.GetNode("c")
	if _, ok := c.OutboundEdges["b"]; ok {
		t.Fatal("outbound edge to removed service should be deleted")
	}
}

func TestSyncServicesKeepsOnlyActive(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	g := NewCallGraph(window, step)

	base := time.Now().Truncate(step)

	// create graph: x->y and y->z, and a standalone service w
	if err := g.AddEdgeMetric("x", "y", base, EdgeLatency95, 1); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}
	if err := g.AddEdgeMetric("y", "z", base, EdgeLatency95, 2); err != nil {
		t.Fatalf("AddEdgeMetric error: %v", err)
	}
	// create service w with at least one metric
	if err := g.AddServiceMetric("w", base, ServiceRequestCount, 1); err != nil {
		t.Fatalf("AddServiceMetric error: %v", err)
	}

	// keep only y
	g.SyncServices([]string{"y"})

	if _, ok := g.GetNode("y"); !ok {
		t.Fatal("y should remain after sync")
	}
	if _, ok := g.GetNode("x"); ok {
		t.Fatal("x should be removed by sync")
	}
	if _, ok := g.GetNode("z"); ok {
		t.Fatal("z should be removed by sync")
	}
	if _, ok := g.GetNode("w"); ok {
		t.Fatal("w should be removed by sync")
	}

	// ensure y has no edges to non-active peers
	y, _ := g.GetNode("y")
	if len(y.OutboundEdges) != 0 || len(y.InboundEdges) != 0 {
		t.Fatalf("expected y edges cleaned, got in=%d out=%d", len(y.InboundEdges), len(y.OutboundEdges))
	}
}
