package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PeronGH/virtcap/internal/dxgi"
)

var fallbackEncoders = []string{
	"hevc_mf",
	"libx265",
}

var vendorPrimaryEncoders = map[dxgi.Vendor]string{
	dxgi.VendorNVIDIA: "hevc_nvenc",
	dxgi.VendorAMD:    "hevc_amf",
	dxgi.VendorIntel:  "hevc_qsv",
}

var vendorPreference = []dxgi.Vendor{
	dxgi.VendorNVIDIA,
	dxgi.VendorAMD,
	dxgi.VendorIntel,
}

type Runner interface {
	Probe(ctx context.Context, ffmpegPath string, adapterIndex int, outputIndex int, encoder string, probeGrace time.Duration, stderr io.Writer) error
}

type ExecRunner struct{}

func SelectEncoder(
	ctx context.Context,
	ffmpegPath string,
	adapterIndex int,
	outputIndex int,
	vendor dxgi.Vendor,
	probeGrace time.Duration,
	runner Runner,
	stderr io.Writer,
	verbose bool,
) (string, error) {
	candidates := EncoderCandidatesForVendor(vendor)
	failures := make([]string, 0, len(candidates))

	for _, encoder := range candidates {
		if verbose {
			fmt.Fprintf(stderr, "probing encoder %s\n", encoder)
		}

		if err := runner.Probe(ctx, ffmpegPath, adapterIndex, outputIndex, encoder, probeGrace, stderr); err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}

			failures = append(failures, fmt.Sprintf("%s: %v", encoder, err))

			if verbose {
				fmt.Fprintf(stderr, "encoder %s failed: %v\n", encoder, err)
			}

			continue
		}

		return encoder, nil
	}

	return "", fmt.Errorf("no encoder probe succeeded: %s", strings.Join(failures, "; "))
}

func EncoderCandidatesForVendor(vendor dxgi.Vendor) []string {
	ordered := make([]string, 0, len(vendorPrimaryEncoders)+len(fallbackEncoders))
	seen := make(map[string]struct{}, len(vendorPrimaryEncoders)+len(fallbackEncoders))

	if primary, ok := vendorPrimaryEncoders[vendor]; ok {
		ordered = append(ordered, primary)
		seen[primary] = struct{}{}
	}

	for _, candidateVendor := range vendorPreference {
		encoder := vendorPrimaryEncoders[candidateVendor]
		if _, ok := seen[encoder]; ok {
			continue
		}

		ordered = append(ordered, encoder)
		seen[encoder] = struct{}{}
	}

	for _, encoder := range fallbackEncoders {
		if _, ok := seen[encoder]; ok {
			continue
		}

		ordered = append(ordered, encoder)
	}

	return ordered
}

func (ExecRunner) Probe(
	ctx context.Context,
	ffmpegPath string,
	adapterIndex int,
	outputIndex int,
	encoder string,
	probeGrace time.Duration,
	stderr io.Writer,
) error {
	timeout := probeGrace + 5*time.Second
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, ffmpegPath, BuildProbeArgs(adapterIndex, outputIndex, encoder, probeGrace)...)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if probeCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("probe timed out")
		}

		return err
	}

	return nil
}

func StartCapture(
	ctx context.Context,
	ffmpegPath string,
	adapterIndex int,
	outputIndex int,
	encoder string,
	stdout io.Writer,
	stderr io.Writer,
) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, ffmpegPath, BuildCaptureArgs(adapterIndex, outputIndex, encoder)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

func BuildProbeArgs(adapterIndex int, outputIndex int, encoder string, probeGrace time.Duration) []string {
	args := buildBaseArgs(adapterIndex, outputIndex, encoder)
	args = append(args,
		"-t", formatSeconds(probeGrace),
		"-f", "null", "-",
	)

	return args
}

func BuildCaptureArgs(adapterIndex int, outputIndex int, encoder string) []string {
	args := buildBaseArgs(adapterIndex, outputIndex, encoder)
	args = append(args,
		"-f", "hevc", "pipe:1",
	)

	return args
}

func buildBaseArgs(adapterIndex int, outputIndex int, encoder string) []string {
	return []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-nostdin",
		"-init_hw_device", fmt.Sprintf("d3d11va=cap:%d", adapterIndex),
		"-filter_hw_device", "cap",
		"-filter_complex", buildFilterGraph(outputIndex, encoder),
		"-an",
		"-c:v", encoder,
	}
}

func buildFilterGraph(outputIndex int, encoder string) string {
	graph := fmt.Sprintf("ddagrab=output_idx=%d", outputIndex)
	if encoder == "libx265" {
		return graph + ",hwdownload,format=bgra,format=yuv420p"
	}

	return graph
}

func formatSeconds(duration time.Duration) string {
	return strconv.FormatFloat(duration.Seconds(), 'f', 3, 64)
}
