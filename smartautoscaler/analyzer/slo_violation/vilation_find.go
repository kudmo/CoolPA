package sloviolation

import (
	"log/slog"
	"math"
	"time"

	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/graph"
	"github.com/kudmo/CoolPA/storage/metrics"
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
func findAbnormalCalls(now time.Time, p AbnormalParams, g *graph.CallGraph) []call {
	fromTime := now.Add(-p.Window)
	threshold := p.SLO * (1 - p.Alpha)

	var abnormalCalls []call

	for _, from := range g.GetServices() {
		node, _ := g.GetNode(from)
		for to, edge := range node.OutboundEdges {
			lat := edge.Latency95.AvgRange(fromTime, now)
			if lat > threshold {
				abnormalCalls = append(abnormalCalls, call{from, to})
			}
		}
	}

	return abnormalCalls
}

// buildCorrelationGraphFromCalls builds a correlation graph from provided abnormal
// calls and service anomaly degrees.
func buildCorrelationGraphFromCalls(now time.Time, p AbnormalParams, s *storage.Storage, abnormalCalls []call, serviceAnomaly map[string]float64) (*types.CorrelationGraph, error) {
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
		srcNode, _ := s.Graph.GetNode(c.from)

		edge := srcNode.OutboundEdges[c.to]

		latSeries := edge.Latency95.SeriesRange(fromTime, now)

		weight := 0.0
		for m_id := 0; m_id < int(metrics.MetricCount); m_id++ {
			m, _, err := s.ResourceMetrics.GetServiceMetricAvgSeries(c.to, metrics.MetricID(m_id), fromTime, now)
			if err != nil {
				continue
			}
			weight = math.Max(weight, pearson(latSeries, m))
		}

		if weight < 0 {
			weight = -weight
		}

		if err := cg.AddEdge(c.from, c.to, weight); err != nil {
			return nil, err
		}
	}

	return cg, nil
}

func computeAnomalyDegree(now time.Time, p AbnormalParams, g *graph.CallGraph, abnormalCalls []call) map[string]float64 {
	fromTime := now.Add(-p.Window)
	serviceAnomaly := make(map[string]float64)

	affectedServices := make(map[string]bool)
	for _, c := range abnormalCalls {
		affectedServices[c.from] = true
		affectedServices[c.to] = true
	}

	for svc := range affectedServices {
		node, exists := g.GetNode(svc)
		if !exists {
			continue
		}

		exceedCount := 0

		for _, edge := range node.InboundEdges {
			latencySeries := edge.Latency95.SeriesRange(fromTime, now)
			for _, lat := range latencySeries {
				if lat > p.SLO {
					exceedCount++
				}
			}

		}

		serviceAnomaly[svc] = float64(exceedCount)
	}

	return serviceAnomaly
}
func BuildAbnormalCorrelationGraph(
	now time.Time,
	p AbnormalParams,
	s *storage.Storage,
) (*types.CorrelationGraph, error) {
	abnormalCalls := findAbnormalCalls(now, p, s.Graph)
	slog.Info("Finded abnormal calls", "count", len(abnormalCalls))
	serviceAnomaly := computeAnomalyDegree(now, p, s.Graph, abnormalCalls)
	slog.Info("Calculatet anomaly degrees", "services", serviceAnomaly)

	return buildCorrelationGraphFromCalls(now, p, s, abnormalCalls, serviceAnomaly)
}
