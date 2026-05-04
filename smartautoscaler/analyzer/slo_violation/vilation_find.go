package sloviolation

import (
	"log/slog"
	"math"
	"time"

	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/graph"
	"github.com/kudmo/CoolPA/storage/metrics"
	"github.com/kudmo/CoolPA/utils"
	"github.com/kudmo/toporank/types"
)

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

	var anomalous []call

	for _, from := range g.GetServices() {
		node, _ := g.GetNode(from)
		for to, edge := range node.OutboundEdges {
			lat95 := edge.Latency95.AvgRange(fromTime, now)
			lat50 := edge.Latency50.AvgRange(fromTime, now)
			if lat95 > p.SLO*(1-p.Alpha) {
				anomalous = append(anomalous, call{from, to})
			} else if lat95/lat50 > 12 {
				anomalous = append(anomalous, call{from, to})
			}
		}
	}
	return anomalous
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
			weight = math.Max(weight, utils.Pearson(latSeries, m))
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
	for _, a := range serviceAnomaly {
		if a != 0 {
			return buildCorrelationGraphFromCalls(now, p, s, abnormalCalls, serviceAnomaly)
		}
	}
	return nil, nil
}
