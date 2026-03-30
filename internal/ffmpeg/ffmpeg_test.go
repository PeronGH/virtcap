package ffmpeg

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestSelectEncoderUsesFirstSuccessfulCandidate(t *testing.T) {
	runner := &fakeRunner{
		failures: map[string]error{
			"hevc_amf": errors.New("amf failed"),
			"hevc_qsv": nil,
		},
	}

	got, err := SelectEncoder(context.Background(), "ffmpeg", 1, 2, time.Second, runner, io.Discard, false)
	if err != nil {
		t.Fatalf("SelectEncoder() error = %v", err)
	}

	if got != "hevc_qsv" {
		t.Fatalf("SelectEncoder() = %q, want hevc_qsv", got)
	}

	if strings.Join(runner.seen, ",") != "hevc_amf,hevc_qsv" {
		t.Fatalf("probe order = %v, want [hevc_amf hevc_qsv]", runner.seen)
	}
}

func TestSelectEncoderReturnsJoinedFailure(t *testing.T) {
	runner := &fakeRunner{
		failures: map[string]error{
			"hevc_amf":   errors.New("amf failed"),
			"hevc_qsv":   errors.New("qsv failed"),
			"hevc_nvenc": errors.New("nvenc failed"),
			"hevc_mf":    errors.New("mf failed"),
			"libx265":    errors.New("libx265 failed"),
		},
	}

	_, err := SelectEncoder(context.Background(), "ffmpeg", 0, 0, time.Second, runner, io.Discard, false)
	if err == nil {
		t.Fatalf("SelectEncoder() error = nil, want failure")
	}

	for _, token := range []string{"hevc_amf", "hevc_qsv", "hevc_nvenc", "hevc_mf", "libx265"} {
		if !strings.Contains(err.Error(), token) {
			t.Fatalf("SelectEncoder() error = %q, want token %q", err.Error(), token)
		}
	}
}

func TestBuildProbeArgsUsesHardwareDeviceAndNullSink(t *testing.T) {
	args := BuildProbeArgs(3, 1, "hevc_nvenc", 1500*time.Millisecond)
	got := strings.Join(args, " ")

	for _, token := range []string{
		"-init_hw_device d3d11va=cap:3",
		"-filter_hw_device cap",
		"-filter_complex ddagrab=output_idx=1",
		"-c:v hevc_nvenc",
		"-t 1.500",
		"-f null -",
	} {
		if !strings.Contains(got, token) {
			t.Fatalf("BuildProbeArgs() = %q, want token %q", got, token)
		}
	}
}

func TestBuildCaptureArgsSoftwareFallbackDownloadsFrames(t *testing.T) {
	args := BuildCaptureArgs(0, 2, "libx265")
	got := strings.Join(args, " ")

	for _, token := range []string{
		"-filter_complex ddagrab=output_idx=2,hwdownload,format=bgra,format=yuv420p",
		"-c:v libx265",
		"-f hevc pipe:1",
	} {
		if !strings.Contains(got, token) {
			t.Fatalf("BuildCaptureArgs() = %q, want token %q", got, token)
		}
	}
}

type fakeRunner struct {
	failures map[string]error
	seen     []string
}

func (f *fakeRunner) Probe(_ context.Context, _ string, _ int, _ int, encoder string, _ time.Duration, _ io.Writer) error {
	f.seen = append(f.seen, encoder)
	if err, ok := f.failures[encoder]; ok {
		return err
	}

	return nil
}
