package web

import (
	"encoding/json"
	"math"
)

type ServiceNode struct {
	ServiceNamespace string   `json:"service_namespace"`
	ServiceName      string   `json:"service_name"`
	OutboundEdges    []string `json:"outbound_edges"`
}

type NullableFloat float64

func (n NullableFloat) MarshalJSON() ([]byte, error) {
	f := float64(n)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return []byte("null"), nil
	}
	return json.Marshal(f)
}

type HistogramBucket struct {
	BoundUpper         NullableFloat `json:"bound_upper"`
	BoundRealValue     float64       `json:"bound_real_value"`
	BoundInterpolation float64       `json:"bound_interpolation"`
}

type ServiceHistogram struct {
	ServiceID string            `json:"service_id"`
	Buckets   []HistogramBucket `json:"buckets"`
}

type PageData struct {
	Graph      []ServiceNode      `json:"graph"`
	Histograms []ServiceHistogram `json:"histograms"`
}
