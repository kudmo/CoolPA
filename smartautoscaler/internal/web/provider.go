package web

import (
	"context"

	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/statistics"
)

type ScalerDataProvider struct {
	MetricsProvider metrics.MetricsRepository
	HistStore       *statistics.HistStore
}

func (p *ScalerDataProvider) GetGraph(ctx context.Context) []ServiceNode {
	services, _ := p.MetricsProvider.ListServices(ctx)
	nodes := make([]ServiceNode, 0)

	if len(services) == 0 {
		return nodes
	}

	visited := make(map[string]bool)
	stack := make([]string, 0, len(services))

	stack = append(stack, services...)

	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[curr] {
			continue
		}
		visited[curr] = true

		svcInfo, err := p.MetricsProvider.GetService(ctx, curr)

		if err != nil {
			continue
		}

		for _, in := range svcInfo.InboundCalls {
			if !visited[in] {
				stack = append(stack, in)
			}
		}

		for _, out := range svcInfo.OutboundCalls {
			if !visited[out] {
				stack = append(stack, out)
			}
		}

		nodes = append(nodes, ServiceNode{
			ServiceName:   curr,
			OutboundEdges: svcInfo.OutboundCalls,
		})
	}

	return nodes
}

func (p *ScalerDataProvider) GetHistograms(ctx context.Context) []ServiceHistogram {
	services, _ := p.MetricsProvider.ListServices(ctx)

	res := make([]ServiceHistogram, 0)
	for _, s := range services {
		h := histogramMapper(p.HistStore.GetHistogram(s))
		h.ServiceID = s
		res = append(res, h)
	}
	return res
}

func histogramMapper(h *statistics.Histogram) ServiceHistogram {
	res := ServiceHistogram{}
	if h == nil {
		return res
	}

	for i := range h.Bins {
		bin := &h.Bins[i]
		real := float64(0.0)
		total, viol := bin.Snapshot()
		if total > 0 {
			real = float64(viol) / float64(total)
		}
		res.Buckets = append(res.Buckets, HistogramBucket{
			BoundUpper:         NullableFloat(bin.UpperBound),
			BoundRealValue:     real,
			BoundInterpolation: h.Risk(bin.UpperBound - 1),
		})
	}
	return res
}
