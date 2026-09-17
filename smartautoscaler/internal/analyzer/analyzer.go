package analyzer

import (
	"context"
)

type Analyzer interface {
	Analyze(ctx context.Context, scalingNamespace string) AnalysisResult
}
