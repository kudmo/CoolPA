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

	g.AddServiceSample("svc1", base, 5, 123.4, 1000, 2000)

	n, ok := g.GetNode("svc1")
	if !ok {
		t.Fatal("expected node svc1 to exist")
	}

	if got := n.RequestCount.Sum(); got != 5 {
		t.Fatalf("RequestCount.Sum() = %v, want 5", got)
	}
	if got := n.RequestDuration.Sum(); got != 123.4 {
		t.Fatalf("RequestDuration.Sum() = %v, want 123.4", got)
	}
	if got := n.BytesSent.Sum(); got != 1000 {
		t.Fatalf("BytesSent.Sum() = %v, want 1000", got)
	}
	if got := n.BytesReceived.Sum(); got != 2000 {
		t.Fatalf("BytesReceived.Sum() = %v, want 2000", got)
	}

	// adding another delta should accumulate for RequestCount and bytes
	g.AddServiceSample("svc1", base, 2, 10, 5, 10) // duration will overwrite bucket
	if got := n.RequestCount.Sum(); got != 7 {
		t.Fatalf("after second sample RequestCount.Sum() = %v, want 7", got)
	}
}

func TestAddEdgeSampleCreatesBothSides(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	g := NewCallGraph(window, step)

	base := time.Now().Truncate(step)

	g.AddEdgeSample("svc-a", "svc-b", base, 50, 20)

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

	g.AddEdgeSample("a", "b", base, 1, 1)
	g.AddEdgeSample("c", "b", base, 2, 2)

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

	g.AddEdgeSample("x", "y", base, 1, 1)
	g.AddEdgeSample("y", "z", base, 2, 2)
	g.AddServiceSample("w", base, 1, 1, 1, 1)

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
