package storage

import (
	"math"
	"sync"
)

type Edge struct {
	Src, Dst string
	Weight   float64
}

type CorrelationGraph struct {
	lock  sync.RWMutex
	edges map[string]map[string]float64
	nodes map[string]struct{}
}

func NewCorrelationGraph() *CorrelationGraph {
	return &CorrelationGraph{
		edges: make(map[string]map[string]float64),
		nodes: make(map[string]struct{}),
	}
}

func (g *CorrelationGraph) AddEdge(src, dst string, weight float64) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.nodes[src] = struct{}{}
	g.nodes[dst] = struct{}{}

	if _, ok := g.edges[src]; !ok {
		g.edges[src] = make(map[string]float64)
	}
	g.edges[src][dst] = weight
}

func (g *CorrelationGraph) NodeList() []string {
	g.lock.RLock()
	defer g.lock.RUnlock()

	nodes := make([]string, 0, len(g.nodes))
	for n := range g.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func Pearson(xs, ys []float64) float64 {
	if len(xs) != len(ys) || len(xs) == 0 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	n := float64(len(xs))
	for i := range xs {
		x := xs[i]
		y := ys[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}
	num := sumXY - (sumX*sumY)/n
	den := math.Sqrt((sumX2 - sumX*sumX/n) * (sumY2 - sumY*sumY/n))
	if den == 0 {
		return 0
	}
	return num / den
}

func BuildCorrelationGraph(
	appMetrics *AppMetricsStore,
	structuralGraph map[string][]string,
	metricName string,
	threshold float64) *CorrelationGraph {

	cg := NewCorrelationGraph()

	for src, neighbors := range structuralGraph {
		srcValues := appMetrics.GetValues(src, metricName)
		if srcValues == nil || len(srcValues) == 0 {
			continue
		}
		for _, dst := range neighbors {
			dstValues := appMetrics.GetValues(dst, metricName)
			if dstValues == nil || len(dstValues) == 0 {
				continue
			}
			corr := Pearson(srcValues, dstValues)
			if math.Abs(corr) >= threshold {
				cg.AddEdge(src, dst, corr)
			}
		}
	}

	return cg
}
