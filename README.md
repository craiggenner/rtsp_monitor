# rtsp-monitor

A Go binary that periodically probes RTSP camera streams and exposes Prometheus metrics, so you can alert when a camera goes offline.

## Build

```bash
go build -o rtsp-monitor .
```

## Configuration

Copy and edit the example config:

```bash
cp config.example.yaml config.yaml
```

See [config.example.yaml](config.example.yaml) for all options with documentation.

### Key settings

| Setting | Default | Description |
|---|---|---|
| `cameras.<name>.url` | *(required)* | Stream URL (`rtsp://`, `rtsps://`, `http://`, or `https://`) |
| `cameras.<name>.interval` | `60` | Seconds between probes |
| `cameras.<name>.timeout` | `30` | Probe timeout in seconds |
| `cameras.<name>.screenshots` | `false` | Save a JPEG screenshot per probe |
| `settings.metric_prefix` | `rtsp` | Prometheus metric name prefix |
| `settings.log_level` | `error` | `debug`, `info`, `warn`, `error` |
| `settings.screenshots_dir` | *(empty)* | Directory for screenshots (required if any camera has `screenshots: true`) |
| `settings.port` | `2112` | HTTP port for `/metrics` endpoint |

## Usage

```bash
./rtsp-monitor --config config.yaml
```

Metrics are served at `http://localhost:2112/metrics` (or your configured port).

### Prometheus metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `<prefix>_success` | Gauge | `camera` | `1` if the stream probe succeeded, `0` if it failed |
| `<prefix>_duration_seconds` | Gauge | `camera` | Time taken for the probe (connect + 5s read) |

### Example Prometheus alert

```yaml
groups:
  - name: cameras
    rules:
      - alert: CameraOffline
        expr: rtsp_success == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Camera {{ $labels.camera }} is offline"
```

## How it works

1. On startup, a goroutine is launched per camera.
2. Each goroutine probes the RTSP stream at the configured interval.
3. A probe connects to the stream URL, reads for 5 seconds, then disconnects.
4. For RTSP URLs, the probe uses **gortsplib** (pure Go) by default, falling back to **ffmpeg**. For HTTP/HTTPS URLs (e.g. Reolink FLV), **ffmpeg** is used directly.
5. When `screenshots: true`, ffmpeg is preferred (it can capture JPEG frames). Screenshots are saved to `<screenshots_dir>/<camera_name>/YYYY/MM/DD/HH:MM.jpg`.
6. On `SIGINT`/`SIGTERM`, in-flight probes are cancelled and the process exits gracefully.

## Smoke testing

You can test locally using [MediaMTX](https://github.com/bluenviron/mediamtx) (a zero-config RTSP server) and ffmpeg:

```bash
# 1. Start a local RTSP server
docker run --rm -p 8554:8554 bluenviron/mediamtx:latest

# 2. Publish a test stream (generates a colour bars pattern)
ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 \
  -c:v libx264 -preset ultrafast -tune zerolatency \
  -f rtsp rtsp://localhost:8554/test

# 3. Create a test config
cat > config.yaml <<'EOF'
cameras:
  test_cam:
    url: "rtsp://localhost:8554/test"
    interval: 15
    timeout: 10
    screenshots: false

settings:
  metric_prefix: "rtsp"
  log_level: "info"
  port: 2112
EOF

# 4. Run the monitor
go run . --config config.yaml

# 5. Check metrics (in another terminal)
curl -s http://localhost:2112/metrics | grep rtsp_
# Expected output:
#   rtsp_duration_seconds{camera="test_cam"} <value>
#   rtsp_success{camera="test_cam"} 1
```

To test screenshot capture, set `screenshots: true` and `screenshots_dir` to a local directory, then ensure `ffmpeg` is on your `PATH`.

To test failure detection, stop the ffmpeg publisher (step 2) and wait for the next probe — `rtsp_success` should flip to `0`.
