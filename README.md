# virtcap

`virtcap` is a Windows-only Go CLI that creates one Parsec virtual display, keeps it alive, resolves the new monitor to a DXGI output, then launches `ffmpeg` and writes raw HEVC bytes to stdout.

## Requirements

- Windows with the Parsec VDD driver already installed and healthy.
- `ffmpeg` available in `PATH`, or pass `--ffmpeg`.

## Usage

```sh
virtcap [--ffmpeg ffmpeg] [--match-timeout 10s] [--probe-grace 2s] [--preset 1080p|1200p|1440p|4k|3k|2.8k|1600p|uwqhd|uw1600p|dual-1080p|3:2-medium|3:2-large|surface-pro] [--verbose]
```

Behavior:

- Opens the Parsec VDD device.
- Adds one virtual display.
- Applies an optional named display preset before capture.
- Sends `VDD_IOCTL_UPDATE` every 100 ms until shutdown.
- Waits for exactly one new Parsec display to appear.
- Matches that display to the DXGI adapter/output pair that Windows assigned to it.
- Probes the matched GPU's preferred hardware encoder first when possible, then falls back through the remaining hardware vendors, `hevc_mf`, and finally `libx265`.
- Uses low-latency encoder settings and writes raw HEVC to stdout.
- Avoids duplicating unchanged desktop frames and preserves varying frame timing to reduce bandwidth when the screen is mostly static.

Preset mappings:

- `1080p` = `1920x1080@60`
- `1200p` = `1920x1200@60`
- `1440p` = `2560x1440@60`
- `4k` = `3840x2160@60`
- `3k` = `3200x1800@60`
- `2.8k` = `2880x1800@60`
- `1600p` = `2560x1600@60`
- `uwqhd` = `3440x1440@60`
- `uw1600p` = `3840x1600@60`
- `dual-1080p` = `3840x1080@60`
- `3:2-medium` = `2256x1504@60`
- `3:2-large` = `2496x1664@60`
- `surface-pro` = `2736x1824@60`

All diagnostics go to stderr so stdout stays clean for the HEVC stream.

## Build

Linux development and CI can still validate the repo:

```sh
go test ./...
GOOS=windows GOARCH=amd64 go build ./...
```
