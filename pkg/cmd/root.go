package cmd

import (
	"github.com/spf13/cobra"
)

func Execute() {
	rootCmd := newRoot()
	cobra.CheckErr(rootCmd.Execute())
}

func newRoot() *cobra.Command {
	opts := &globalOptions{}

	rootCmd := &cobra.Command{
		Use:   "kubectl-foren",
		Short: "A kubectl plugin for forensic operations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	fs := rootCmd.PersistentFlags()

	fs.BoolVarP(&opts.Verbose,
		longFlagName(opts, "Verbose"),
		shortFlagName(opts, "Verbose"),
		false,
		"verbose output")

	fs.BoolVarP(&opts.Debug,
		longFlagName(opts, "Debug"),
		shortFlagName(opts, "Debug"),
		false,
		"debug output with stacktrace")

	fs.StringVarP(&opts.LogFormat,
		longFlagName(opts, "LogFormat"),
		shortFlagName(opts, "LogFormat"),
		"text",
		"format for logging")

	rootCmd.AddCommand(nodeProcessCmd(fs))
	return rootCmd
}
