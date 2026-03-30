//go:build windows

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/PeronGH/virtcap/internal/display"
	"github.com/PeronGH/virtcap/internal/dxgi"
	"github.com/PeronGH/virtcap/internal/ffmpeg"
	"github.com/PeronGH/virtcap/internal/vdd"
)

const (
	parsecVDDInstallerURL = "https://builds.parsec.app/vdd/parsec-vdd-0.45.0.0.exe"
)

func run(cfg Config, stdout io.Writer, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log := logger{enabled: cfg.Verbose, w: stderr}

	status, instanceID, err := vdd.QueryStatus()
	if err != nil {
		return fmt.Errorf("query Parsec VDD status: %w", err)
	}

	if status == vdd.StatusNotInstalled {
		return fmt.Errorf(
			"parsec VDD is not installed. Download it here: %s",
			parsecVDDInstallerURL,
		)
	}

	if status != vdd.StatusOK {
		return fmt.Errorf("parsec VDD device status is %s", status)
	}

	device, err := vdd.Open(instanceID)
	if err != nil {
		return fmt.Errorf("open Parsec VDD device: %w", err)
	}
	defer device.Close()

	addedIndex, err := device.AddDisplay()
	if err != nil {
		return fmt.Errorf("add Parsec display: %w", err)
	}

	log.Printf("added Parsec display index %d", addedIndex)

	keepaliveCtx, stopKeepalive := context.WithCancel(ctx)
	keepaliveDone := make(chan struct{})
	go func() {
		defer close(keepaliveDone)

		if err := vdd.RunKeepAlive(keepaliveCtx, device, 100*time.Millisecond); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(stderr, "virtcap: keepalive error: %v\n", err)
		}
	}()

	defer func() {
		stopKeepalive()
		<-keepaliveDone
	}()

	newDisplay, err := waitForDisplayIndex(ctx, cfg.MatchTimeout, addedIndex)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	}

	log.Printf("matched Parsec display %s to device %s", newDisplay.InterfaceName, newDisplay.DeviceName)

	outputs, err := dxgi.EnumerateOutputs()
	if err != nil {
		return fmt.Errorf("enumerate DXGI outputs: %w", err)
	}

	output, err := dxgi.MatchOutputByDeviceName(outputs, newDisplay.DeviceName)
	if err != nil {
		return fmt.Errorf("match DXGI output for %s: %w", newDisplay.DeviceName, err)
	}

	log.Printf("resolved DXGI adapter %d output %d (%s)", output.AdapterIndex, output.OutputIndex, output.Vendor)

	encoder, err := ffmpeg.SelectEncoder(
		ctx,
		cfg.FFmpegPath,
		output.AdapterIndex,
		output.OutputIndex,
		output.Vendor,
		cfg.ProbeGrace,
		ffmpeg.ExecRunner{},
		stderr,
		cfg.Verbose,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	}

	log.Printf("selected encoder %s", encoder)

	cmd, err := ffmpeg.StartCapture(ctx, cfg.FFmpegPath, output.AdapterIndex, output.OutputIndex, encoder, cfg.StdoutFormat, stdout, stderr)
	if err != nil {
		return fmt.Errorf("start ffmpeg capture: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}

		return fmt.Errorf("ffmpeg exited with error: %w", err)
	}

	return nil
}

func waitForDisplayIndex(ctx context.Context, timeout time.Duration, displayIndex int) (display.Snapshot, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error

	for {
		displays, err := display.EnumerateParsecDisplays()
		if err != nil {
			lastErr = err
		} else {
			snapshot, matchErr := display.FindDisplayByIndex(displays, display.ParsecDisplayCode, displayIndex)
			if matchErr == nil {
				return snapshot, nil
			}

			lastErr = matchErr
		}

		select {
		case <-ctx.Done():
			return display.Snapshot{}, ctx.Err()
		case <-deadline.C:
			if lastErr == nil {
				lastErr = errors.New("no new Parsec display observed")
			}

			return display.Snapshot{}, fmt.Errorf("wait for Parsec display index %d: %w", displayIndex, lastErr)
		case <-ticker.C:
		}
	}
}

type logger struct {
	enabled bool
	w       io.Writer
}

func (l logger) Printf(format string, args ...any) {
	if !l.enabled {
		return
	}

	fmt.Fprintf(l.w, format+"\n", args...)
}
