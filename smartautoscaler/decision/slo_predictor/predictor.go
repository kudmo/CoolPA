package slopredictor

import (
	"fmt"
	"sync"

	"github.com/kudmo/CoolPA/logger"

	ort "github.com/yalue/onnxruntime_go"
)

// Predictor defines a batched predictor that returns per-service risk scores in [0,1].
type Predictor interface {
	PredictBatch(X [][]float64) []float64
}

type LatencyDeltaPredictor struct {
	session *ort.DynamicAdvancedSession
	mu      sync.Mutex
}

var initOnce sync.Once

func initORT() {
	ort.SetSharedLibraryPath("/usr/local/lib/libonnxruntime.so")

	if err := ort.InitializeEnvironment(); err != nil {
		// Log the initialization error before panicking so administrators see the cause.
		logger.Error("slo_predictor", "failed to initialize ONNX env", "error", err)
		panic(fmt.Errorf("failed to initialize ONNX env: %w", err))
	}
}

func NewSLOPredictor(modelPath string, inputSize int) (*LatencyDeltaPredictor, error) {
	initOnce.Do(initORT)

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"float_input"},
		[]string{"variable"},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &LatencyDeltaPredictor{
		session: session,
	}, nil
}

func (p *LatencyDeltaPredictor) PredictBatch(X [][]float64) []float64 {
	if len(X) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	batch := len(X)
	features := len(X[0])

	inputData := make([]float32, 0, batch*features)
	for _, row := range X {
		for _, val := range row {
			inputData = append(inputData, float32(val))
		}
	}

	inputShape := ort.NewShape(int64(batch), int64(features))

	inputTensor, err := ort.NewTensor[float32](inputShape, inputData)
	if err != nil {
		logger.Error("slo_predictor", "input tensor error", "error", err)
		return nil
	}
	defer inputTensor.Destroy()

	outputShape := ort.NewShape(int64(batch), 1)

	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		logger.Error("slo_predictor", "output tensor error", "error", err)
		return nil
	}
	defer outputTensor.Destroy()

	err = p.session.Run([]ort.Value{inputTensor}, []ort.Value{outputTensor})
	if err != nil {
		logger.Error("slo_predictor", "run error", "error", err)
		return nil
	}

	output := outputTensor.GetData()
	result := make([]float64, batch)
	for i := 0; i < batch; i++ {
		result[i] = float64(output[i])
	}

	return result
}

func (p *LatencyDeltaPredictor) Close() {
	if p.session != nil {
		p.session.Destroy()
	}
}
