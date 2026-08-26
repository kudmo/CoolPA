package contextutil

import (
	"context"
	"time"
)

type contextKey string

const (
	analysisTimeKey contextKey = "analysis_time"
)

func WithAnalysisTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, analysisTimeKey, t.Truncate(time.Second))
}

func GetAnalysisTime(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(analysisTimeKey).(time.Time)
	return t, ok
}

func GetTimeRange(ctx context.Context, windowSize time.Duration) (from, to time.Time, ok bool) {
	analysisTime, exists := GetAnalysisTime(ctx)
	if !exists {
		return time.Time{}, time.Time{}, false
	}

	to = analysisTime
	from = analysisTime.Add(-windowSize)

	return from, to, true
}
