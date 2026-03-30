package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

type Config struct {
	FFmpegPath   string
	MatchTimeout time.Duration
	ProbeGrace   time.Duration
	Verbose      bool
}

func Main(args []string, stdout io.Writer, stderr io.Writer) error {
	cfg, err := parseFlags(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}

		return fmt.Errorf("parse flags: %w", err)
	}

	return run(cfg, stdout, stderr)
}

func parseFlags(args []string, stderr io.Writer) (Config, error) {
	cfg := Config{
		FFmpegPath:   "ffmpeg",
		MatchTimeout: 10 * time.Second,
		ProbeGrace:   2 * time.Second,
	}

	fs := flag.NewFlagSet("virtcap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.FFmpegPath, "ffmpeg", cfg.FFmpegPath, "Path to the ffmpeg executable.")
	fs.DurationVar(&cfg.MatchTimeout, "match-timeout", cfg.MatchTimeout, "How long to wait for the new Parsec display to appear.")
	fs.DurationVar(&cfg.ProbeGrace, "probe-grace", cfg.ProbeGrace, "How long each ffmpeg encoder probe runs before it is accepted.")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Write progress logs to stderr.")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [flags]\n\n", fs.Name())
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if fs.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	if cfg.MatchTimeout <= 0 {
		return Config{}, fmt.Errorf("--match-timeout must be greater than zero")
	}

	if cfg.ProbeGrace <= 0 {
		return Config{}, fmt.Errorf("--probe-grace must be greater than zero")
	}

	if strings.TrimSpace(cfg.FFmpegPath) == "" {
		return Config{}, fmt.Errorf("--ffmpeg must not be empty")
	}

	return cfg, nil
}
