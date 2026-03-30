package ffmpeg

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/PeronGH/virtcap/internal/dxgi"
)

func TestSelectEncoderUsesFirstSuccessfulCandidate(t *testing.T) {
	runner := &fakeRunner{
		failures: map[string]error{
			"hevc_nvenc": errors.New("nvenc failed"),
			"hevc_amf":   nil,
		},
	}

	got, err := SelectEncoder(context.Background(), "ffmpeg", 1, 2, dxgi.VendorNVIDIA, time.Second, runner, io.Discard, false)
	if err != nil {
		t.Fatalf("SelectEncoder() error = %v", err)
	}

	if got != "hevc_amf" {
		t.Fatalf("SelectEncoder() = %q, want hevc_amf", got)
	}

	if strings.Join(runner.seen, ",") != "hevc_nvenc,hevc_amf" {
		t.Fatalf("probe order = %v, want [hevc_nvenc hevc_amf]", runner.seen)
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

	_, err := SelectEncoder(context.Background(), "ffmpeg", 0, 0, dxgi.VendorUnknown, time.Second, runner, io.Discard, false)
	if err == nil {
		t.Fatalf("SelectEncoder() error = nil, want failure")
	}

	for _, token := range []string{"hevc_amf", "hevc_qsv", "hevc_nvenc", "hevc_mf", "libx265"} {
		if !strings.Contains(err.Error(), token) {
			t.Fatalf("SelectEncoder() error = %q, want token %q", err.Error(), token)
		}
	}
}

func TestEncoderCandidatesForVendor(t *testing.T) {
	tests := []struct {
		name   string
		vendor dxgi.Vendor
		want   string
	}{
		{name: "nvidia", vendor: dxgi.VendorNVIDIA, want: "hevc_nvenc,hevc_amf,hevc_qsv,hevc_mf,libx265"},
		{name: "amd", vendor: dxgi.VendorAMD, want: "hevc_amf,hevc_nvenc,hevc_qsv,hevc_mf,libx265"},
		{name: "intel", vendor: dxgi.VendorIntel, want: "hevc_qsv,hevc_nvenc,hevc_amf,hevc_mf,libx265"},
		{name: "unknown", vendor: dxgi.VendorUnknown, want: "hevc_nvenc,hevc_amf,hevc_qsv,hevc_mf,libx265"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.Join(EncoderCandidatesForVendor(tt.vendor), ",")
			if got != tt.want {
				t.Fatalf("EncoderCandidatesForVendor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildProbeArgsUsesHardwareDeviceAndNullSink(t *testing.T) {
	args := BuildProbeArgs(3, 1, "hevc_nvenc", 1500*time.Millisecond)
	got := strings.Join(args, " ")

	for _, token := range []string{
		"-init_hw_device d3d11va=cap:3",
		"-filter_hw_device cap",
		"-filter_complex ddagrab=output_idx=1:dup_frames=0",
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
	args := BuildCaptureArgs(0, 2, "libx265", OutputFormatMPEGTS)
	got := strings.Join(args, " ")

	for _, token := range []string{
		"-filter_complex ddagrab=output_idx=2:dup_frames=0,hwdownload,format=bgra,format=yuv420p",
		"-flags +low_delay",
		"-g 30",
		"-bf 0",
		"-tune zerolatency",
		"-x265-params repeat-headers=1:keyint=30:min-keyint=30",
		"-c:v libx265",
		"-fps_mode passthrough",
		"-flush_packets 1",
		"-muxdelay 0",
		"-muxpreload 0",
		"-mpegts_flags pat_pmt_at_frames",
		"-f mpegts pipe:1",
	} {
		if !strings.Contains(got, token) {
			t.Fatalf("BuildCaptureArgs() = %q, want token %q", got, token)
		}
	}
}

func TestBuildCaptureArgsRawHEVCKeepsElementaryStream(t *testing.T) {
	args := BuildCaptureArgs(1, 0, "hevc_nvenc", OutputFormatHEVC)
	got := strings.Join(args, " ")

	for _, token := range []string{
		"-preset llhq",
		"-tune ull",
		"-fps_mode passthrough",
		"-f hevc pipe:1",
	} {
		if !strings.Contains(got, token) {
			t.Fatalf("BuildCaptureArgs() = %q, want token %q", got, token)
		}
	}
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		value   string
		want    OutputFormat
		wantErr bool
	}{
		{value: "mpegts", want: OutputFormatMPEGTS},
		{value: "HEVC", want: OutputFormatHEVC},
		{value: "bad", wantErr: true},
	}

	for _, tt := range tests {
		got, err := ParseOutputFormat(tt.value)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseOutputFormat(%q) error = nil, want error", tt.value)
			}

			continue
		}

		if err != nil {
			t.Fatalf("ParseOutputFormat(%q) error = %v", tt.value, err)
		}

		if got != tt.want {
			t.Fatalf("ParseOutputFormat(%q) = %q, want %q", tt.value, got, tt.want)
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
