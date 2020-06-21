package main

import (
	"fmt"
	"io"

	"github.com/cnabio/cnab-go/bundle/loader"
	"github.com/spf13/cobra"

	"github.com/cnabio/duffle/pkg/duffle/home"
	"github.com/cnabio/duffle/pkg/imagestore/construction"
	"github.com/cnabio/duffle/pkg/packager"
)

const exportDesc = `
Packages a bundle, and by default any images referenced by the bundle, within a single gzipped tarfile.

Unless --thin is specified, a thick bundle is exported. A thick bundle contains the bundle manifest and all images
(including invocation images) referenced by the bundle metadata. Images are saved as an OCI image layout in the
artifacts/layout/ directory.

If --thin specified, only the bundle manifest is exported.

By default, this command will use the name and version information of the bundle to create a compressed archive file
called <name>-<version>.tgz in the current directory. This destination can be updated by specifying a file path to save
the compressed bundle to using the --output-file flag.

A path to a bundle file may be passed in instead of a bundle in local storage by using the --bundle-is-file flag, thus:
$ duffle export [PATH] --bundle-is-file
`

type exportCmd struct {
	bundle       string
	dest         string
	home         home.Home
	out          io.Writer
	thin         bool
	verbose      bool
	bundleIsFile bool
}

func newExportCmd(w io.Writer) *cobra.Command {
	export := &exportCmd{out: w}

	cmd := &cobra.Command{
		Use:   "export [BUNDLE]",
		Short: "package CNAB bundle in gzipped tar file",
		Long:  exportDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			export.home = home.Home(homePath())
			export.bundle = args[0]

			return export.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&export.dest, "output-file", "o", "", "Save exported bundle to file path")
	f.BoolVarP(&export.bundleIsFile, "bundle-is-file", "f", false, "Interpret the bundle source as a file path")
	f.BoolVarP(&export.thin, "thin", "t", false, "Export only the bundle manifest")
	f.BoolVarP(&export.verbose, "verbose", "v", false, "Verbose output")

	return cmd
}

func (ex *exportCmd) run() error {
	bundlefile, l, err := ex.setup()
	if err != nil {
		return err
	}
	if err := ex.Export(bundlefile, l); err != nil {
		return err
	}

	return nil
}

func (ex *exportCmd) Export(bundlefile string, l loader.BundleLoader) error {
	ctor, err := construction.NewConstructor(ex.thin)
	if err != nil {
		return err
	}

	exp, err := packager.NewExporter(bundlefile, ex.dest, ex.home.Logs(), l, ctor)
	if err != nil {
		return fmt.Errorf("Unable to set up exporter: %s", err)
	}
	if err := exp.Export(); err != nil {
		return err
	}
	if ex.verbose {
		fmt.Fprintf(ex.out, "Export logs: %s\n", exp.Logs())
	}
	return nil
}

func (ex *exportCmd) setup() (string, loader.BundleLoader, error) {
	l := loader.New()
	bundlefile, err := resolveBundleFilePath(ex.bundle, ex.home.String(), ex.bundleIsFile)
	if err != nil {
		return "", l, err
	}

	return bundlefile, l, nil
}

func resolveBundleFilePath(bun, homePath string, bundleIsFile bool) (string, error) {
	if bundleIsFile {
		return bun, nil
	}

	bundlefile, err := getBundleFilepath(bun, homePath)
	if err != nil {
		return "", err
	}
	return bundlefile, err
}
