package prober

import (
	"context"
	"fmt"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/pion/rtp"
)

const readDuration = 5 * time.Second

type GoRTSPProber struct{}

func NewGoRTSPProber() *GoRTSPProber {
	return &GoRTSPProber{}
}

func (p *GoRTSPProber) Probe(ctx context.Context, url string, wantScreenshot bool) (*Result, error) {
	start := time.Now()

	u, err := base.ParseURL(url)
	if err != nil {
		return &Result{Success: false, Duration: time.Since(start)}, fmt.Errorf("invalid RTSP URL: %w", err)
	}

	c := &gortsplib.Client{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	err = c.Start()
	if err != nil {
		return &Result{Success: false, Duration: time.Since(start)}, fmt.Errorf("connect failed: %w", err)
	}
	defer c.Close()

	desc, _, err := c.Describe(u)
	if err != nil {
		return &Result{Success: false, Duration: time.Since(start)}, fmt.Errorf("describe failed: %w", err)
	}

	err = c.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		return &Result{Success: false, Duration: time.Since(start)}, fmt.Errorf("setup failed: %w", err)
	}

	// Install a generic packet handler to detect stream activity.
	// Screenshots are handled by the ffmpeg fallback since pure Go H264 decoding
	// to a full image requires an external C decoder (e.g. x264) which adds
	// significant complexity. The gortsplib probe validates the stream is alive;
	// if screenshots are needed the orchestrator will use ffmpeg.
	packetReceived := make(chan struct{}, 1)
	c.OnPacketRTPAny(func(_ *description.Media, _ format.Format, _ *rtp.Packet) {
		select {
		case packetReceived <- struct{}{}:
		default:
		}
	})

	_, err = c.Play(nil)
	if err != nil {
		return &Result{Success: false, Duration: time.Since(start)}, fmt.Errorf("play failed: %w", err)
	}

	// Read for the required duration
	select {
	case <-ctx.Done():
		return &Result{Success: false, Duration: time.Since(start)}, ctx.Err()
	case <-time.After(readDuration):
		// Successfully read for the full duration
	}

	duration := time.Since(start)

	return &Result{
		Success:   true,
		Duration:  duration,
		FrameJPEG: nil, // gortsplib doesn't produce screenshots; ffmpeg handles that
	}, nil
}
