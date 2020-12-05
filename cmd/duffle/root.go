package main

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var verbose bool

// newRootCmd builds the root duffle command
// - outputRedirect: Optional, specify to capture all command output (stderr and stdout)
func newRootCmd(outputRedirect io.Writer) *cobra.Command {
	const usage = `The CNAB installer`

	cmd := &cobra.Command{
		Use:          "duffle",
		Short:        usage,
		SilenceUsage: true,
		Long:         usage,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if verbose {
				log.SetLevel(log.DebugLevel)
			}
			if cmd.Name() == "init" || cmd.Name() == "version" {
				return nil
			}
			err := autoInit(cmd.OutOrStdout(), false)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "pre-flight check failed: %s", err)
			}
			return err
		},
	}
	cmd.SetOutput(outputRedirect)
	outLog := cmd.OutOrStdout()

	p := cmd.PersistentFlags()
	p.StringVar(&duffleHome, "home", defaultDuffleHome(), "location of your Duffle config. Overrides $DUFFLE_HOME")
	p.BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	cmd.AddCommand(
		newBuildCmd(outLog),
		newBundleCmd(outLog),
		newInitCmd(outLog),
		newListCmd(outLog),
		newRelocateCmd(outLog),
		newVersionCmd(outLog),
		newInstallCmd(outLog),
		newStatusCmd(outLog),
		newUninstallCmd(outLog),
		newUpgradeCmd(outLog),
		newRunCmd(outLog),
		newCredentialsCmd(outLog),
		newClaimsCmd(outLog),
		newExportCmd(outLog),
		newImportCmd(outLog),
		newCreateCmd(outLog),
	)

	return cmd
}
