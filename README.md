# virtcap

`virtcap` is a Windows-only Go CLI that creates one Parsec virtual display, keeps it alive, resolves the new monitor to a DXGI output, then launches `ffmpeg` and writes raw HEVC bytes to stdout.

## Requirements

- Windows with the Parsec VDD driver already installed and healthy.
- `ffmpeg` available in `PATH`, or pass `--ffmpeg`.

## Usage

```sh
virtcap [--ffmpeg ffmpeg] [--match-timeout 10s] [--probe-grace 2s] [--verbose]
```

Behavior:

- Opens the Parsec VDD device.
- Adds one virtual display.
- Sends `VDD_IOCTL_UPDATE` every 100 ms until shutdown.
- Waits for exactly one new Parsec display to appear.
- Matches that display to the DXGI adapter/output pair that Windows assigned to it.
- Probes the matched GPU's preferred hardware encoder first when possible, then falls back through the remaining hardware vendors, `hevc_mf`, and finally `libx265`.
- Uses low-latency encoder settings and writes raw HEVC to stdout.
- Avoids duplicating unchanged desktop frames and preserves varying frame timing to reduce bandwidth when the screen is mostly static.

All diagnostics go to stderr so stdout stays clean for the HEVC stream.

## Build

Linux development and CI can still validate the repo:

```sh
go test ./...
GOOS=windows GOARCH=amd64 go build ./...
```
