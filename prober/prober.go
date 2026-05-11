package prober

import (
	"context"
	"time"
)

// Result holds the outcome of a single camera probe.
type Result struct {
	Success   bool
	Duration  time.Duration
	FrameJPEG []byte // nil when screenshots not requested or on failure
}

// Prober can probe an RTSP stream.
type Prober interface {
	Probe(ctx context.Context, url string, wantScreenshot bool) (*Result, error)
}
