package analyzer

import (
	"context"
)

type Analyzer interface {
	Analyze(ctx context.Context) AnalysisResult
}
