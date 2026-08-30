package analyzer

import (
	"context"
	"math"
	"sort"

	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/utils"
	"github.com/kudmo/toporank/api"
	"github.com/kudmo/toporank/types"
)

// call represents a directed edge between two services.
type call struct {
	from string
	to   string
}

// findAbnormalCalls identifies service calls that are abnormal based on
// latency thresholds. A call is abnormal if its average P95 latency exceeds
// the SLO adjusted by Alpha, or if the ratio of P95 to P50 latency is high
// (indicating tail latency issues). Returns the list of abnormal calls.
func (a *Analyzer) findAbnormalCalls(ctx context.Context) []call {
	var anomalous []call

	services, _ := a.metricsProvider.ListServices(ctx)
	for _, from_name := range services {
		from, _ := a.metricsProvider.GetService(ctx, from_name)
		for _, to_name := range from.OutboundCalls {
			lat95_range, _ := a.metricsProvider.GetGraphLatencyP95Range(ctx, from_name, to_name, a.config.Window)
			lat95 := utils.Avg(lat95_range)
			lat50_range, _ := a.metricsProvider.GetGraphLatencyP50Range(ctx, from_name, to_name, a.config.Window)
			lat50 := utils.Avg(lat50_range)

			if lat95 > a.config.SLO*(1-a.config.Alpha) {
				anomalous = append(anomalous, call{from_name, to_name})
			} else if lat95/lat50 > 12 {
				anomalous = append(anomalous, call{from_name, to_name})
			}
		}
	}
	return anomalous
}

// buildCorrelationGraphFromCalls constructs a correlation graph from abnormal
// calls and per‑service anomaly degrees. Edge weights are computed as the
// maximum absolute Pearson correlation between the destination service’s
// latency and various resource usage metrics. Returns nil if no abnormal calls.
func (a *Analyzer) buildCorrelationGraphFromCalls(ctx context.Context, abnormalCalls []call, serviceAnomaly map[string]float64) (*types.CorrelationGraph, error) {
	if len(abnormalCalls) == 0 {
		return nil, nil
	}

	cg := types.NewCorrelationGraph()
	for svc, degree := range serviceAnomaly {
		if err := cg.AddNode(svc, degree); err != nil {
			return nil, err
		}
	}

	for _, c := range abnormalCalls {
		latSeries, _ := a.metricsProvider.GetGraphLatencyP95Range(ctx, c.from, c.to, a.config.Window)
		weight := 0.0

		// Compute max correlation across several resource metrics.
		cpuUsage, err := a.metricsProvider.GetServiceCpuUsageRange(ctx, c.to, a.config.Window)
		if err == nil {
			weight = math.Max(weight, math.Abs(utils.Pearson(latSeries, cpuUsage)))
		}
		memUsage, err := a.metricsProvider.GetServiceMemoryUsageRange(ctx, c.to, a.config.Window)
		if err == nil {
			weight = math.Max(weight, math.Abs(utils.Pearson(latSeries, memUsage)))
		}
		fsUsage, err := a.metricsProvider.GetServiceFSUsageRange(ctx, c.to, a.config.Window)
		if err == nil {
			weight = math.Max(weight, math.Abs(utils.Pearson(latSeries, fsUsage)))
		}
		fsWrite, err := a.metricsProvider.GetServiceFSWriteRange(ctx, c.to, a.config.Window)
		if err == nil {
			weight = math.Max(weight, math.Abs(utils.Pearson(latSeries, fsWrite)))
		}
		fsRead, err := a.metricsProvider.GetServiceFSReadRange(ctx, c.to, a.config.Window)
		if err == nil {
			weight = math.Max(weight, math.Abs(utils.Pearson(latSeries, fsRead)))
		}
		netRecv, err := a.metricsProvider.GetServiceNetworkReceiveRange(ctx, c.to, a.config.Window)
		if err == nil {
			weight = math.Max(weight, math.Abs(utils.Pearson(latSeries, netRecv)))
		}
		netTrans, err := a.metricsProvider.GetServiceNetworkTransmitRange(ctx, c.to, a.config.Window)
		if err == nil {
			weight = math.Max(weight, math.Abs(utils.Pearson(latSeries, netTrans)))
		}

		if err := cg.AddEdge(c.from, c.to, weight); err != nil {
			return nil, err
		}
	}

	return cg, nil
}

// computeAnomalyDegree calculates a simple anomaly score for each service
// involved in abnormal calls. The score is the total number of latency
// samples (from inbound calls) that exceed the SLO. Only services that
// participate in abnormal calls (as source or destination) are considered.
func (a *Analyzer) computeAnomalyDegree(ctx context.Context, abnormalCalls []call) map[string]float64 {
	serviceAnomaly := make(map[string]float64)

	affectedServices := make(map[string]bool)
	for _, c := range abnormalCalls {
		affectedServices[c.from] = true
		affectedServices[c.to] = true
	}

	for svc := range affectedServices {
		node, exists := a.metricsProvider.GetService(ctx, svc)
		if exists != nil {
			continue
		}

		exceedCount := 0
		for _, from := range node.InboundCalls {
			latencySeries, _ := a.metricsProvider.GetGraphLatencyP95Range(ctx, from, node.Name, a.config.Window)
			for _, lat := range latencySeries {
				if lat > a.config.SLO {
					exceedCount++
				}
			}
		}
		serviceAnomaly[svc] = float64(exceedCount)
	}

	return serviceAnomaly
}

// buildAbnormalCorrelationGraph orchestrates the detection of abnormal calls,
// computation of anomaly degrees, and construction of the correlation graph.
// It returns nil if no anomaly degree is greater than zero.
func (a *Analyzer) buildAbnormalCorrelationGraph(ctx context.Context) (*types.CorrelationGraph, error) {
	abnormalCalls := a.findAbnormalCalls(ctx)
	logger.Info("slo_violation", "found abnormal calls", "count", len(abnormalCalls))
	serviceAnomaly := a.computeAnomalyDegree(ctx, abnormalCalls)
	logger.Info("slo_violation", "calculated anomaly degrees", "services", serviceAnomaly)

	for _, anomaly := range serviceAnomaly {
		if anomaly != 0 {
			return a.buildCorrelationGraphFromCalls(ctx, abnormalCalls, serviceAnomaly)
		}
	}
	return nil, nil
}

// analyzeWithSLOViolation detects services that violate their SLO using a
// correlation graph and the TopoRank algorithm. It returns the names of the
// top anomalous services (up to AnomalyServicesCount), excluding the
// istio-ingressgateway if present. Returns an empty slice if no anomalies are found.
func (a *Analyzer) analyzeWithSLOViolation(ctx context.Context) []string {
	result := make([]string, 0)

	graph, err := a.buildAbnormalCorrelationGraph(ctx)
	if err != nil {
		logger.Error("analyzer", "failed to build abnormal correlation graph", "error", err)
	} else {
		if graph == nil {
			return result
		}
	}
	anomalys := api.RunTopoRank(graph, types.DefaultConfig())
	sort.Slice(anomalys, func(i, j int) bool {
		return anomalys[i].Rank > anomalys[j].Rank
	})

	// Exclude the ingress gateway from scaling recommendations.
	for i := 0; i < len(anomalys) && len(result) < a.config.AnomalyServicesCount; i++ {
		if anomalys[i].ID != "istio-ingressgateway" {
			result = append(result, anomalys[i].ID)
		}
	}

	for _, service := range anomalys {
		logger.Info("analyzer", "calculated anomaly", "service", service.ID, "value", service.Rank)
	}
	return result
}
