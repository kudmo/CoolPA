package slopredictor

import (
	"log/slog"

	ort "github.com/yalue/onnxruntime_go"
)

// Predictor defines a batched predictor that returns per-service risk scores in [0,1].
type Predictor interface {
	PredictBatch(X [][]float64) []float64
}

type SLOPredictor struct{}

func (d *SLOPredictor) PredictBatch(X [][]float64) []float64 {
	out := make([]float64, len(X))

	ort.SetSharedLibraryPath("/usr/local/lib/libonnxruntime.so")

	err := ort.InitializeEnvironment()
	if err != nil {
		slog.Error("Failed inicialize predictor", "error", err)
	}
	defer ort.DestroyEnvironment()

	for i, x := range X {

		inputData := x

		inputShape := ort.NewShape(1, int64(len(inputData)))

		inputTensor, err := ort.NewTensor[float64](inputShape, inputData)
		if err != nil {
			slog.Error("Failed inicialize inputTensor", "error", err)
		}
		defer inputTensor.Destroy()

		outputShape := ort.NewShape(1, 2)

		outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
		if err != nil {
			slog.Error("Failed inicialize outputTensor", "error", err)
		}
		defer outputTensor.Destroy()

		session, err := ort.NewAdvancedSession(
			"random_forest_model.onnx",
			[]string{"double_input"},
			[]string{"probabilities"},
			[]ort.ArbitraryTensor{inputTensor},
			[]ort.ArbitraryTensor{outputTensor},
			nil,
		)

		if err != nil {
			slog.Error("Failed inicialize predictor session", "error", err)
		}
		defer session.Destroy()

		err = session.Run()
		if err != nil {
			slog.Error("Failed run predictor", "error", err)
		}

		outputData := outputTensor.GetData()
		out[i] = float64(outputData[1]) // Вероятность попадения в класс 1 == нарушение SLO
	}

	return out
}
