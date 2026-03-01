package sloviolation

import (
	"math"
	"time"

	"github.com/kudmo/CoolPA/storage/graph"
	"github.com/kudmo/toporank/types"
)

func pearson(x, y []float64) float64 {
	n := len(x)
	if n == 0 || len(y) != n {
		return 0
	}

	var sumX, sumY float64
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
	}

	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var num, denX, denY float64
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		num += dx * dy
		denX += dx * dx
		denY += dy * dy
	}

	if denX == 0 || denY == 0 {
		return 0
	}

	return num / math.Sqrt(denX*denY)
}

type AbnormalParams struct {
	Window time.Duration
	SLO    float64
	Alpha  float64
}

type call struct {
	from string
	to   string
}

// findAbnormalCalls detects abnormal calls according to parameters and returns
// the list of abnormal edges along with an anomaly degree per service.
func findAbnormalCalls(now time.Time, p AbnormalParams, g *graph.CallGraph) ([]call, map[string]float64) {
	fromTime := now.Add(-p.Window)
	threshold := p.SLO * (1 + p.Alpha/2)

	var abnormalCalls []call
	serviceAnomaly := make(map[string]float64)

	for _, from := range g.GetServices() {
		node, _ := g.GetNode(from)
		for to, edge := range node.OutboundEdges {
			lat := edge.Latency95.AvgRange(fromTime, now)

			if lat > threshold {
				abnormalCalls = append(abnormalCalls, call{from, to})

				// anomaly degree = normalized exceed ratio
				excess := (lat - threshold) / threshold
				if excess < 0 {
					excess = 0
				}
				if excess > 1 {
					excess = 1
				}

				// assign to both services (max if multiple edges)
				if excess > serviceAnomaly[from] {
					serviceAnomaly[from] = excess
				}
				if excess > serviceAnomaly[to] {
					serviceAnomaly[to] = excess
				}
			}
		}
	}

	return abnormalCalls, serviceAnomaly
}

// buildCorrelationGraphFromCalls builds a correlation graph from provided abnormal
// calls and service anomaly degrees.
func buildCorrelationGraphFromCalls(now time.Time, p AbnormalParams, g *graph.CallGraph, abnormalCalls []call, serviceAnomaly map[string]float64) (*types.CorrelationGraph, error) {
	if len(abnormalCalls) == 0 {
		return nil, nil
	}

	fromTime := now.Add(-p.Window)

	cg := types.NewCorrelationGraph()
	for svc, degree := range serviceAnomaly {
		if err := cg.AddNode(svc, degree); err != nil {
			return nil, err
		}
	}

	for _, c := range abnormalCalls {
		srcNode, _ := g.GetNode(c.from)
		dstNode, _ := g.GetNode(c.to)

		edge := srcNode.OutboundEdges[c.to]

		latSeries := edge.Latency95.SeriesRange(fromTime, now)
		reqSeries := dstNode.RequestCount.SeriesRange(fromTime, now)

		weight := pearson(latSeries, reqSeries)

		if weight < 0 {
			weight = -weight
		}

		if err := cg.AddEdge(c.from, c.to, weight); err != nil {
			return nil, err
		}
	}

	return cg, nil
}

func BuildAbnormalCorrelationGraph(
	now time.Time,
	p AbnormalParams,
	g *graph.CallGraph,
) (*types.CorrelationGraph, error) {
	abnormalCalls, serviceAnomaly := findAbnormalCalls(now, p, g)
	return buildCorrelationGraphFromCalls(now, p, g, abnormalCalls, serviceAnomaly)
}
