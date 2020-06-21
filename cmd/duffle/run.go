package main

import (
	"fmt"
	"io"
	"log"

	"github.com/spf13/cobra"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/claim"
)

type runCmd struct {
	action     string
	claimName  string
	bundleName string
	bundlePath string

	relocationMapping string
	credentialsFiles  []string
	valuesFile        string
	setParams         []string
	setFiles          []string

	driver    string
	out       io.Writer
	opOutFunc action.OperationConfigFunc

	home    string
	storage *claim.Store
	claim   *claim.Claim
}

func newRunCmd(w io.Writer) *cobra.Command {
	const short = "run an action in the bundle"
	const long = `Run an arbitrary action in the bundle.

Some CNAB bundles may declare custom actions in addition to install, upgrade, and uninstall.
This command can be used to execute those actions.

The 'run' command takes an ACTION and a CLAIM name:

  $ duffle run migrate --claim myExistingClaim
or
  $ duffle run preinstall --bundle myBundle --claim myNewClaim
or
  $ duffle run preinstall --bundle-is-file path/to/bundle.json --claim myNewClaim

All custom actions can be executed on claims (installed bundles).
Stateless custom actions can be executed on claims or bundles.
A new claim will be created if a bundle is specified and the action modifies.

Credentials and parameters may also be passed in.
`
	run := &runCmd{out: w}
	cmd := &cobra.Command{
		Use:     "run ACTION --claim CLAIM [--bundle BUNDLE | --bundle-is-file path/to/bundle.json]",
		Aliases: []string{"exec"},
		Short:   short,
		Long:    long,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			run.action = args[0]
			run.home = homePath()
			return run.run()
		},
	}
	run.opOutFunc = setOut(cmd.OutOrStdout())

	flags := cmd.Flags()
	flags.StringVarP(&run.claimName, "claim", "i", "", "Specify the name of an existing claim (required)")
	flags.StringVarP(&run.bundleName, "bundle", "b", "", "Specify the name of a bundle")
	flags.StringVarP(&run.bundlePath, "bundle-is-file", "f", "", "Specify the path to a bundle.json")
	flags.StringVarP(&run.driver, "driver", "d", "docker", "Specify a driver name")
	flags.StringVarP(&run.relocationMapping, "relocation-mapping", "m", "", "Path of relocation mapping JSON file")
	flags.StringArrayVarP(&run.credentialsFiles, "credentials", "c", []string{}, "Specify a set of credentials to use inside the CNAB bundle")
	flags.StringVarP(&run.valuesFile, "parameters", "p", "", "Specify file containing parameters. Formats: toml, MORE SOON")
	flags.StringArrayVarP(&run.setParams, "set", "s", []string{}, "Set individual parameters as NAME=VALUE pairs")

	err := cmd.MarkFlagRequired("claim")
	if err != nil {
		log.Fatal("required flag \"claim\" is missing")
	}

	return cmd
}

func (r *runCmd) run() error {
	if r.storage == nil {
		storage := claimStorage()
		r.storage = &storage
	}

	err := r.prepareClaim()
	if err != nil {
		return fmt.Errorf("failed to prepare claim %q: %s", r.claimName, err)
	}

	creds, err := loadCredentials(r.credentialsFiles, r.claim.Bundle)
	if err != nil {
		return err
	}

	driver, err := prepareDriver(r.driver)
	if err != nil {
		return fmt.Errorf("failed to prepare driver %q: %s", r.driver, err)
	}

	// Override parameters only if some are set.
	if r.valuesFile != "" || len(r.setParams) > 0 {
		r.claim.Parameters, err = calculateParamValues(r.claim.Bundle, r.valuesFile, r.setParams, r.setFiles)
		if err != nil {
			return fmt.Errorf("failed to set parameters on claim: %s", err)
		}
	}

	opRelocator, err := makeOpRelocator(r.relocationMapping)
	if err != nil {
		return err
	}

	action := &action.RunCustom{
		Driver: driver,
		Action: r.action,
	}

	fmt.Fprintf(r.out, "Executing custom action %q\n", r.action)
	err = action.Run(r.claim, creds, r.opOutFunc, opRelocator)
	if actionDef := r.claim.Bundle.Actions[r.action]; !actionDef.Modifies {
		// Do not store a claim for non-mutating actions.
		return err
	}

	storageErr := r.storage.Store(*r.claim)
	if err != nil {
		return fmt.Errorf("run failed: %s", err)
	}

	return storageErr
}

func (r *runCmd) prepareClaim() error {
	var err error

	if r.bundleName != "" && r.bundlePath != "" {
		return fmt.Errorf("cannot specify both --bundle and --bundle-is-file: received bundle %q and bundle file %q", r.bundleName, r.bundlePath)
	}

	if r.bundleName != "" {
		r.bundlePath, err = resolveBundleFilePath(r.bundleName, r.home, false)
		if err != nil {
			return err
		}
	}

	if r.bundlePath != "" {
		return r.createClaimFromBundlePath()
	}

	return r.useExistingClaim()
}

func (r *runCmd) createClaimFromBundlePath() error {
	if !fileExists(r.bundlePath) {
		return fmt.Errorf("bundle file %q does not exist", r.bundlePath)
	}

	bundle, err := loadBundle(r.bundlePath)
	if err != nil {
		return fmt.Errorf("failed to parse contents in bundle file %q: %s", r.bundlePath, err)
	}

	r.claim, err = claim.New(r.claimName)
	if err != nil {
		return fmt.Errorf("failed to create claim %q: %s", r.claimName, err)
	}

	r.claim.Bundle = bundle
	return nil
}

func (r *runCmd) useExistingClaim() error {
	c, err := r.storage.Read(r.claimName)
	if err != nil {
		if err == claim.ErrClaimNotFound {
			return fmt.Errorf("claim %q not found in duffle store", r.claimName)
		}
		return fmt.Errorf("failed to read claim %q from duffle store: %s", r.claimName, err)
	}

	r.claim = &c
	return nil
}
