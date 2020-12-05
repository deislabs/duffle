package main

import (
	"io"

	"github.com/cnabio/duffle/pkg/duffle"
	"github.com/cnabio/duffle/pkg/duffle/home"

	"github.com/spf13/cobra"
)

const bundleRemoveDesc = `Remove a bundle from the local storage.

This removes a bundle from the local storage so that it will no longer be locally
available. Bundles can be rebuilt with 'duffle build'.

Ex. $ duffle bundle remove foo  # removes all versions of foo from local store

If a SemVer range is provided with '--version'/'-r' then only releases that match
that range will be removed.
`

type bundleRemoveCmd struct {
	bundleRef string
	home      home.Home
	out       io.Writer
	versions  string
}

func newBundleRemoveCmd(w io.Writer) *cobra.Command {
	remove := &bundleRemoveCmd{out: w}

	cmd := &cobra.Command{
		Use:     "remove [BUNDLE]",
		Aliases: []string{"rm"},
		Short:   "remove a bundle from the local storage",
		Long:    bundleRemoveDesc,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			remove.bundleRef = args[0]
			remove.home = home.Home(homePath())

			return remove.run()
		},
	}
	cmd.Flags().StringVarP(&remove.versions, "version", "r", "", "A version or SemVer2 version range")

	return cmd
}

func (rm *bundleRemoveCmd) run() error {
	return duffle.Remove(rm.out, rm.bundleRef, rm.home, rm.versions)
}
