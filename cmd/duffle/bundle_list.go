package main

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/cnabio/duffle/pkg/duffle"
	"github.com/cnabio/duffle/pkg/duffle/home"
)

func newBundleListCmd(w io.Writer) *cobra.Command {
	var short bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list bundles pulled or built and stored locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			return duffle.List(w, home.Home(homePath()), short)
		},
	}
	cmd.Flags().BoolVarP(&short, "short", "s", false, "output shorter listing format")
	return cmd
}
