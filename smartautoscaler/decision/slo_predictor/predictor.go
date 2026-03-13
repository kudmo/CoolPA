package slopredictor

import (
	"fmt"
	"log"

	ort "github.com/yalue/onnxruntime_go"
)

// DummyPredictor is a simple rule-based predictor used as a placeholder.
type RFPredictor struct{}

func (d *RFPredictor) PredictBatch(X [][]float64) []float64 {
	out := make([]float64, len(X))

	ort.SetSharedLibraryPath("/usr/local/lib/libonnxruntime.so")

	err := ort.InitializeEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	defer ort.DestroyEnvironment()

	// --- 3. Подготовьте входные и выходные тензоры ---
	// Данные для предсказания. Должны быть в формате []float32.
	// Здесь для примера массив из 20 признаков.
	for i, x := range X {

		inputData := x

		inputShape := ort.NewShape(1, int64(len(inputData)))
		inputTensor, err := ort.NewTensor(inputShape, inputData)
		if err != nil {
			log.Fatal(err)
		}
		defer inputTensor.Destroy()

		// Определите форму выходного тензора.
		// Для классификации это может быть (1, количество_классов) для вероятностей или (1,) для метки.
		// Пока мы не знаем точный размер, можно создать пустой тензор с предположительной формой.
		// Важно: если форма не совпадет с тем, что вернет модель, Run() вернет ошибку.
		// Лучше всего заранее узнать форму выхода, например, из Python.
		outputShape := ort.NewShape(1)
		outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
		if err != nil {
			log.Fatal(err)
		}
		defer outputTensor.Destroy()

		// --- 4. Создайте сессию для инференса ---
		// Здесь мы используем AdvancedSession.
		// Первый аргумент — путь к .onnx файлу.
		// Второй — список имен входных слоев (их можно узнать из Python, например, sess.get_inputs()[0].name)
		// Третий — список имен выходных слоев.
		// Четвертый и пятый — списки входных и выходных тензоров.
		session, err := ort.NewAdvancedSession("/app/random_forest_model.onnx",
			[]string{"float_input"},
			[]string{"output_label"},
			[]ort.Value{inputTensor},
			[]ort.Value{outputTensor},
			nil)
		if err != nil {
			log.Fatal(err)
		}
		defer session.Destroy()

		err = session.Run()
		if err != nil {
			log.Fatal(err)
		}

		outputData := outputTensor.GetData()
		fmt.Println("Предсказание модели в Go:", outputData)
		out[i] = float64(outputData[0])
	}

	return out
}
