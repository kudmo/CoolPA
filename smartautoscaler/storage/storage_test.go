package storage

import (
	"testing"
	"time"

	"github.com/kudmo/CoolPA/storage/metrics"
)

func TestNewStorageInit(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	s := NewStorage(window, step)
	if s == nil {
		t.Fatal("NewStorage returned nil")
	}
	if s.Graph == nil {
		t.Fatal("Graph not initialized")
	}
	if s.ResourceMetrics == nil {
		t.Fatal("ResourceMetrics not initialized")
	}
}

func TestAddResourceSampleAndInvalidMetric(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	s := NewStorage(window, step)

	svc := "svc-res"
	pod := "pod-1"
	ts := time.Now()

	// valid metric
	err := s.AddResourceSample(svc, pod, metrics.CPUUsage, ts, 42)
	if err != nil {
		t.Fatalf("AddResourceSample returned error: %v", err)
	}

	pods := s.ResourceMetrics.GetServicePods(svc)
	if pods == nil || len(pods) == 0 {
		t.Fatalf("service %s not created in ResourceMetrics", svc)
	}
	// check bucket value at head via getter
	if got, ok, err := s.ResourceMetrics.GetPodMetricHeadValue(svc, pod, metrics.CPUUsage); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !ok {
		t.Fatalf("pod %s not created", pod)
	} else if got != 42 {
		t.Fatalf("expected metric value 42, got %v", got)
	}

	// invalid metric
	err = s.AddResourceSample(svc, pod, metrics.MetricID(99), ts, 1)
	if err == nil {
		t.Fatal("expected error for invalid metric, got nil")
	}
	if err != metrics.ErrInvalidMetric {
		t.Fatalf("expected ErrInvalidMetric, got %v", err)
	}
}

func TestAddIstioServiceAndEdgeSamples(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	s := NewStorage(window, step)

	ts := time.Now().Truncate(step)

	s.AddIstioServiceSample("svc-1", ts, 3, 99.9, 1000, 2000)

	n, ok := s.Graph.GetNode("svc-1")
	if !ok {
		t.Fatal("svc-1 node missing")
	}
	if got := n.RequestCount.Sum(); got != 3 {
		t.Fatalf("expected RequestCount 3, got %v", got)
	}
	if got := n.RequestDuration.Sum(); got != 99.9 {
		t.Fatalf("expected RequestDuration 99.9, got %v", got)
	}

	// edge sample
	s.AddIstioEdgeSample("a", "b", ts, 55, 22)
	src, ok := s.Graph.GetNode("a")
	if !ok {
		t.Fatal("a missing")
	}
	out, ok := src.OutboundEdges["b"]
	if !ok {
		t.Fatal("expected outbound edge a->b")
	}
	if got := out.Latency95.Sum(); got != 55 {
		t.Fatalf("expected latency95 55, got %v", got)
	}

	dst, ok := s.Graph.GetNode("b")
	if !ok {
		t.Fatal("b missing")
	}
	in, ok := dst.InboundEdges["a"]
	if !ok {
		t.Fatal("expected inbound edge a->b on b")
	}
	if got := in.Latency50.Sum(); got != 22 {
		t.Fatalf("expected latency50 22, got %v", got)
	}
}

func TestSyncRemovesGraphNodesAndSyncsPods(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second
	s := NewStorage(window, step)

	ts := time.Now().Truncate(step)

	// resource pods for svcA
	s.AddResourceSample("svcA", "pod1", metrics.CPUUsage, ts, 1)
	s.AddResourceSample("svcA", "pod2", metrics.CPUUsage, ts, 2)

	// graph services
	s.AddIstioServiceSample("x", ts, 1, 1, 1, 1)
	s.AddIstioServiceSample("y", ts, 1, 1, 1, 1)
	s.AddIstioServiceSample("z", ts, 1, 1, 1, 1)

	// sync: keep only svcA with pod1 and service y
	active := map[string][]string{
		"svcA": {"pod1"},
		"y":    {},
	}
	s.Sync(active)

	// resource pods: svcA should have only pod1
	pods := s.ResourceMetrics.GetServicePods("svcA")
	if pods == nil {
		t.Fatal("svcA missing in resource metrics after sync")
	}
	if len(pods) != 1 {
		t.Fatalf("expected 1 pod for svcA after sync, got %d", len(pods))
	}
	found := false
	for _, p := range pods {
		if p == "pod1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pod1 should exist after sync")
	}

	// graph: only y should remain
	if _, ok := s.Graph.GetNode("y"); !ok {
		t.Fatal("y should remain in graph after sync")
	}
	if _, ok := s.Graph.GetNode("x"); ok {
		t.Fatal("x should be removed from graph by sync")
	}
	if _, ok := s.Graph.GetNode("z"); ok {
		t.Fatal("z should be removed from graph by sync")
	}
}

func TestNilStorageReceiversDontPanic(t *testing.T) {
	var s *Storage
	// these calls should not panic and should be safe no-ops (or return error for resources)
	// Resource sample on nil receiver should return nil (we implemented early return)
	if err := s.AddResourceSample("svc", "pod", metrics.CPUUsage, time.Now(), 1); err != nil {
		t.Fatalf("expected nil error when calling AddResourceSample on nil receiver, got %v", err)
	}
	// Istio methods should be safe no-ops
	s.AddIstioServiceSample("svc", time.Now(), 1, 1, 1, 1)
	s.AddIstioEdgeSample("a", "b", time.Now(), 1, 1)
	// Sync on nil receiver should not panic
	s.Sync(map[string][]string{"x": {"p"}})
}

func TestSyncWithEmptyActiveRemovesGraphButLeavesResourceMetrics(t *testing.T) {
	s := NewStorage(30*time.Second, 10*time.Second)

	// create resource metric
	s.AddResourceSample("svcR", "p1", metrics.CPUUsage, time.Now(), 5)
	// create graph nodes
	s.AddIstioServiceSample("g1", time.Now(), 1, 1, 1, 1)
	s.AddIstioServiceSample("g2", time.Now(), 1, 1, 1, 1)

	// sync with empty map
	s.Sync(map[string][]string{})

	// ResourceMetrics should still have svcR
	pods := s.ResourceMetrics.GetServicePods("svcR")
	if pods == nil || len(pods) == 0 {
		t.Fatalf("resource service svcR should remain after empty sync, got pods=%v", pods)
	}

	// Graph should be cleared
	if _, ok := s.Graph.GetNode("g1"); ok {
		t.Fatalf("graph node g1 should be removed by empty sync")
	}
	if _, ok := s.Graph.GetNode("g2"); ok {
		t.Fatalf("graph node g2 should be removed by empty sync")
	}
}
