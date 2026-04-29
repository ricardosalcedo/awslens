package cmd

import (
	"log/slog"
	"os"

	"github.com/awslens/awslens/internal/tui"
	"github.com/spf13/cobra"
)

var (
	profile    string
	region     string
	monthsBack int
	debug      bool
)

var rootCmd = &cobra.Command{
	Use:   "awslens",
	Short: "A modern TUI for AWS — see everything, navigate fast",
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging(debug)

		p, r := profile, region

		for {
			if p == "" {
				var err error
				p, r, err = tui.RunPicker()
				if err != nil {
					return err
				}
			}

			if region != "" {
				r = region
			}

			err := tui.Start(p, r, monthsBack)
			if err == tui.ErrBackToPicker {
				p, r = "", ""
				continue
			}
			return err
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "AWS profile (skips picker)")
	rootCmd.PersistentFlags().StringVarP(&region, "region", "r", "", "AWS region override")
	rootCmd.PersistentFlags().IntVarP(&monthsBack, "months-back", "m", 0, "Start costs view N months back (0=current)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging to awslens-debug.log")
}

// initLogging configures structured logging. When debug is true, logs at
// DEBUG level are written to awslens-debug.log; otherwise logging is
// effectively disabled (WARN level, discarded).
func initLogging(debug bool) {
	if debug {
		f, err := os.OpenFile("awslens-debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			// Fall back to no-op if we can't open the log file.
			return
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})))
	} else {
		// Discard all logs when not in debug mode.
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.Level(99)})))
	}
}
