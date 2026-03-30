# virtcap

`virtcap` is a Windows-only Go CLI that creates one Parsec virtual display, keeps it alive, resolves the new monitor to a DXGI output, then launches `ffmpeg` and writes a live-friendly HEVC stream to stdout.

## Requirements

- Windows with the Parsec VDD driver already installed and healthy.
- `ffmpeg` available in `PATH`, or pass `--ffmpeg`.

## Usage

```sh
virtcap [--ffmpeg ffmpeg] [--match-timeout 10s] [--probe-grace 2s] [--stdout-format mpegts|hevc] [--verbose]
```

Behavior:

- Opens the Parsec VDD device.
- Adds one virtual display.
- Sends `VDD_IOCTL_UPDATE` every 100 ms until shutdown.
- Waits for exactly one new Parsec display to appear.
- Matches that display to the DXGI adapter/output pair that Windows assigned to it.
- Probes the matched GPU's preferred hardware encoder first when possible, then falls back through the remaining hardware vendors, `hevc_mf`, and finally `libx265`.
- Uses low-latency encoder settings and writes MPEG-TS to stdout by default.
- Avoids duplicating unchanged desktop frames and preserves varying frame timing to reduce bandwidth when the screen is mostly static.
- Keeps `--stdout-format hevc` available for raw HEVC output when needed.

All diagnostics go to stderr so stdout stays clean for the HEVC stream.

## Live Replay

Default low-latency local replay:

```sh
virtcap | ffplay -fflags nobuffer -flags low_delay -probesize 32 -analyzeduration 0 -framedrop -sync video -
```

If you explicitly request raw HEVC output:

```sh
virtcap --stdout-format hevc | ffplay -f hevc -fflags nobuffer -flags low_delay -probesize 32 -analyzeduration 0 -framedrop -sync video -
```

## Build

Linux development and CI can still validate the repo:

```sh
go test ./...
GOOS=windows GOARCH=amd64 go build ./...
```
