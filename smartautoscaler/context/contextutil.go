// Package contextutil provides utilities for managing and retrieving
// time-related information from context.Context.
//
// The package is primarily used to propagate analysis time information
// through the application's call chain, allowing functions to access
// the timestamp when a particular analysis operation began without
// requiring explicit parameter passing.
package contextutil

import (
	"context"
	"time"
)

// contextKey is an unexported type used as a key for context values.
// Using an unexported type prevents collisions with context keys
// from other packages.
type contextKey string

const (
	// analysisTimeKey is the context key used to store the analysis
	// start time. It is unexported to ensure type safety and prevent
	// accidental key collisions.
	analysisTimeKey contextKey = "analysis_time_begin"
)

// WithAnalysisTime returns a new context derived from ctx that contains
// the specified analysis time. The time is truncated to seconds to
// ensure consistent precision across the application.
//
// This function is typically called at the beginning of an analysis
// operation to record when the analysis started, allowing downstream
// functions to calculate time ranges relative to this moment.
//
// Parameters:
//   - ctx: The parent context
//   - t: The analysis start time (will be truncated to seconds)
//
// Returns:
//   - A new context containing the analysis time
func WithAnalysisTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, analysisTimeKey, t.Truncate(time.Second))
}

// GetAnalysisTime retrieves the analysis time from the context.
//
// Parameters:
//   - ctx: The context containing the analysis time
//
// Returns:
//   - The analysis time if present in the context
//   - A boolean indicating whether the time was found (true) or not (false)
//
// Example:
//
//	if analysisTime, ok := contextutil.GetAnalysisTime(ctx); ok {
//	    fmt.Printf("Analysis started at: %v\n", analysisTime)
//	}
func GetAnalysisTime(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(analysisTimeKey).(time.Time)
	return t, ok
}

// GetTimeRange calculates a time range ending at the analysis time
// stored in the context and extending backwards by the specified
// window size.
//
// This function is useful for querying metrics or logs within a
// specific time window relative to when the analysis began.
//
// Parameters:
//   - ctx: The context containing the analysis time
//   - windowSize: The duration of the time window (e.g., 5*time.Minute
//     for a 5-minute analysis window)
//
// Returns:
//   - from: The start of the time range (analysis time minus window size)
//   - to: The end of the time range (equal to the analysis time)
//   - ok: A boolean indicating whether the analysis time was found
//     in the context (true) or not (false)
//
// If the analysis time is not present in the context, the function
// returns zero time values and false.
//
// Example:
//
//	from, to, ok := contextutil.GetTimeRange(ctx, 10*time.Minute)
//	if ok {
//	    metrics := prometheus.QueryRange(from, to)
//	}
func GetTimeRange(ctx context.Context, windowSize time.Duration) (from, to time.Time, ok bool) {
	analysisTime, exists := GetAnalysisTime(ctx)
	if !exists {
		return time.Time{}, time.Time{}, false
	}

	to = analysisTime
	from = analysisTime.Add(-windowSize)

	return from, to, true
}
