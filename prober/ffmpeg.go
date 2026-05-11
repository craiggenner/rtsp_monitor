package prober

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type FFmpegProber struct{}

func NewFFmpegProber() *FFmpegProber {
	return &FFmpegProber{}
}

func (p *FFmpegProber) Probe(ctx context.Context, url string, wantScreenshot bool) (*Result, error) {
	start := time.Now()

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return &Result{Success: false, Duration: time.Since(start)}, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	if wantScreenshot {
		return p.probeWithScreenshot(ctx, url, start)
	}
	return p.probeOnly(ctx, url, start)
}

func (p *FFmpegProber) probeOnly(ctx context.Context, url string, start time.Time) (*Result, error) {
	// Read stream for 5 seconds, discard output
	args := []string{
		"-rtsp_transport", "tcp",
		"-i", url,
		"-t", "5",
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		return &Result{Success: false, Duration: duration}, fmt.Errorf("ffmpeg probe failed: %w", err)
	}

	return &Result{Success: true, Duration: duration}, nil
}

func (p *FFmpegProber) probeWithScreenshot(ctx context.Context, url string, start time.Time) (*Result, error) {
	tmpDir, err := os.MkdirTemp("", "rtsp-screenshot-*")
	if err != nil {
		return &Result{Success: false, Duration: time.Since(start)}, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	framePath := filepath.Join(tmpDir, "frame.jpg")

	// Read stream for 5 seconds and capture a frame near the end
	args := []string{
		"-rtsp_transport", "tcp",
		"-i", url,
		"-t", "5",
		"-frames:v", "1",
		"-update", "1",
		"-q:v", "2",
		framePath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	err = cmd.Run()
	duration := time.Since(start)

	if err != nil {
		return &Result{Success: false, Duration: duration}, fmt.Errorf("ffmpeg probe failed: %w", err)
	}

	var jpegData []byte
	if data, err := os.ReadFile(framePath); err == nil {
		jpegData = data
	}

	return &Result{
		Success:   true,
		Duration:  duration,
		FrameJPEG: jpegData,
	}, nil
}
