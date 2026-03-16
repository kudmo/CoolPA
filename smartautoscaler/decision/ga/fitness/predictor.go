package fitness

import "math"

// Predictor defines a batched predictor that returns per-service risk scores in [0,1].
type Predictor interface {
	PredictBatch(X [][]float64) []float64
}

// DummyPredictor is a simple rule-based predictor used as a placeholder.
type DummyPredictor struct{}

func (d *DummyPredictor) PredictBatch(X [][]float64) []float64 {
	out := make([]float64, len(X))
	for i, x := range X {
		// use the active delta feature at index 2
		risk := 0.0
		if len(x) >= 3 {
			risk = sigmoid(x[2] * 2.0)
		}
		out[i] = clamp01(risk)
	}
	return out
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
