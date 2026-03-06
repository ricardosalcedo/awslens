package cmd

import (
	"os"

	"github.com/awslens/awslens/internal/tui"
	"github.com/spf13/cobra"
)

var (
	profile string
	region  string
)

var rootCmd = &cobra.Command{
	Use:   "awslens",
	Short: "A modern TUI for AWS — see everything, navigate fast",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, r := profile, region

		// if no profile flag, show interactive picker
		if p == "" {
			var err error
			p, r, err = tui.RunPicker()
			if err != nil {
				return err
			}
		}

		// region flag overrides whatever the profile has
		if region != "" {
			r = region
		}

		return tui.Start(p, r)
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
}
