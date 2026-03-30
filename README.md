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
- Matches that display to a DXGI adapter/output pair.
- Probes `hevc_amf`, `hevc_qsv`, `hevc_nvenc`, `hevc_mf`, then `libx265`.
- Starts `ffmpeg` and pipes raw HEVC to stdout.

All diagnostics go to stderr so stdout stays clean for the HEVC stream.

## Build

Linux development and CI can still validate the repo:

```sh
go test ./...
GOOS=windows GOARCH=amd64 go build ./...
```
