package prober

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go.rtsp-monitor/config"
	"go.rtsp-monitor/metrics"
	"go.rtsp-monitor/screenshot"
)

type Orchestrator struct {
	cfg     *config.Config
	metrics *metrics.Metrics
	goProbe *GoRTSPProber
	ffProbe *FFmpegProber
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func NewOrchestrator(cfg *config.Config, m *metrics.Metrics) *Orchestrator {
	return &Orchestrator{
		cfg:     cfg,
		metrics: m,
		goProbe: NewGoRTSPProber(),
		ffProbe: NewFFmpegProber(),
	}
}

// Start launches a goroutine per camera that probes on the configured interval.
func (o *Orchestrator) Start(ctx context.Context) {
	ctx, o.cancel = context.WithCancel(ctx)

	for name, cam := range o.cfg.Cameras {
		o.wg.Add(1)
		go o.runCamera(ctx, name, cam)
	}
}

// Stop cancels all probes and waits for them to finish.
func (o *Orchestrator) Stop() {
	if o.cancel != nil {
		o.cancel()
	}
	o.wg.Wait()
}

func (o *Orchestrator) runCamera(ctx context.Context, name string, cam *config.CameraConfig) {
	defer o.wg.Done()

	slog.Info("starting camera probe loop", "camera", name, "interval", cam.Interval)

	// Run an initial probe immediately
	o.probeCamera(ctx, name, cam)

	ticker := time.NewTicker(time.Duration(cam.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping camera probe loop", "camera", name)
			return
		case <-ticker.C:
			o.probeCamera(ctx, name, cam)
		}
	}
}

func (o *Orchestrator) probeCamera(ctx context.Context, name string, cam *config.CameraConfig) {
	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(cam.Timeout)*time.Second)
	defer cancel()

	slog.Debug("probing camera", "camera", name, "url", cam.URL)

	var result *Result
	var err error

	isHTTP := strings.HasPrefix(cam.URL, "http://") || strings.HasPrefix(cam.URL, "https://")

	if isHTTP {
		// HTTP/HTTPS URLs (e.g. Reolink FLV) — only ffmpeg can handle these.
		result, err = o.ffProbe.Probe(probeCtx, cam.URL, cam.Screenshots)
	} else if cam.Screenshots {
		// RTSP with screenshots — use ffmpeg (it can capture JPEG frames).
		// Fall back to gortsplib if ffmpeg is unavailable.
		result, err = o.ffProbe.Probe(probeCtx, cam.URL, true)
		if err != nil {
			slog.Warn("ffmpeg probe failed, trying gortsplib (no screenshot)", "camera", name, "error", err)
			result, err = o.goProbe.Probe(probeCtx, cam.URL, false)
		}
	} else {
		// RTSP without screenshots — prefer gortsplib, fall back to ffmpeg.
		result, err = o.goProbe.Probe(probeCtx, cam.URL, false)
		if err != nil {
			slog.Warn("gortsplib probe failed, trying ffmpeg fallback", "camera", name, "error", err)
			result, err = o.ffProbe.Probe(probeCtx, cam.URL, false)
		}
	}

	if err != nil {
		slog.Error("all probers failed", "camera", name, "error", err)
		if result != nil {
			o.metrics.Record(name, false, result.Duration.Seconds())
		} else {
			o.metrics.Record(name, false, 0)
		}
		return
	}

	o.metrics.Record(name, result.Success, result.Duration.Seconds())

	if result.Success {
		slog.Info("probe succeeded", "camera", name, "duration", result.Duration)
	} else {
		slog.Warn("probe failed", "camera", name, "duration", result.Duration)
	}

	// Save screenshot if enabled and we captured a frame
	if cam.Screenshots && result.FrameJPEG != nil && o.cfg.Settings.ScreenshotDir != "" {
		if err := screenshot.Save(o.cfg.Settings.ScreenshotDir, name, time.Now(), result.FrameJPEG); err != nil {
			slog.Error("failed to save screenshot", "camera", name, "error", err)
		} else {
			slog.Info("screenshot saved", "camera", name)
		}
	}
}
