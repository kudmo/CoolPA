package slopredictor

import (
	"fmt"
	"log/slog"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// Predictor defines a batched predictor that returns per-service risk scores in [0,1].
type Predictor interface {
	PredictBatch(X [][]float64) []float64
}

type SLOPredictor struct {
	session *ort.DynamicAdvancedSession
	mu      sync.Mutex
}

var initOnce sync.Once

func initORT() {
	ort.SetSharedLibraryPath("/usr/local/lib/libonnxruntime.so")

	if err := ort.InitializeEnvironment(); err != nil {
		panic(fmt.Errorf("failed to initialize ONNX env: %w", err))
	}
}

func NewSLOPredictor(modelPath string, inputSize int) (*SLOPredictor, error) {
	initOnce.Do(initORT)

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"double_input"},
		[]string{"probabilities"},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &SLOPredictor{
		session: session,
	}, nil
}

func (p *SLOPredictor) PredictBatch(X [][]float64) []float64 {
	if len(X) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	batch := len(X)
	features := len(X[0])

	// flatten
	inputData := make([]float64, 0, batch*features)
	for _, row := range X {
		inputData = append(inputData, row...)
	}

	inputShape := ort.NewShape(int64(batch), int64(features))

	inputTensor, err := ort.NewTensor[float64](inputShape, inputData)
	if err != nil {
		slog.Error("input tensor error", "error", err)
		return nil
	}
	defer inputTensor.Destroy()

	outputShape := ort.NewShape(int64(batch), 2)

	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		slog.Error("output tensor error", "error", err)
		return nil
	}
	defer outputTensor.Destroy()

	err = p.session.Run([]ort.Value{inputTensor}, []ort.Value{outputTensor})
	if err != nil {
		slog.Error("run error", "error", err)
		return nil
	}

	output := outputTensor.GetData()

	result := make([]float64, batch)
	for i := 0; i < batch; i++ {
		result[i] = float64(output[i*2+1]) // класс 1
	}

	return result
}

func (p *SLOPredictor) Close() {
	if p.session != nil {
		p.session.Destroy()
	}
}
