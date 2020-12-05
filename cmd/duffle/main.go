package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/loader"
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/credentials"
	"github.com/cnabio/cnab-go/driver"
	"github.com/cnabio/cnab-go/driver/lookup"
	"github.com/cnabio/cnab-go/utils/crud"
	"github.com/spf13/cobra"

	"github.com/cnabio/duffle/pkg/duffle/home"
	"github.com/cnabio/duffle/pkg/reference"
)

var (
	// duffleHome depicts the home directory where all duffle config is stored.
	duffleHome string
	rootCmd    *cobra.Command
)

func main() {
	rootCmd = newRootCmd(nil)
	must(rootCmd.Execute())
}

func homePath() string {
	return os.ExpandEnv(duffleHome)
}

func defaultDuffleHome() string {
	if home := os.Getenv(home.HomeEnvVar); home != "" {
		return home
	}

	homeEnvPath := os.Getenv("HOME")
	if homeEnvPath == "" && runtime.GOOS == "windows" {
		homeEnvPath = os.Getenv("USERPROFILE")
	}

	return filepath.Join(homeEnvPath, ".duffle")
}

// claimStorage returns a claim store for accessing claims.
func claimStorage() claim.Store {
	h := home.Home(homePath())
	return claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
}

// loadCredentials loads a set of credentials from HOME.
func loadCredentials(files []string, b *bundle.Bundle) (map[string]string, error) {
	creds := map[string]string{}
	if len(files) == 0 {
		return creds, credentials.Validate(creds, b.Credentials)
	}

	credDir := home.Home(homePath()).Credentials() // Credentials directory should exist from duffle init

	// The strategy here is "last one wins". We loop through each credential file and
	// calculate its credentials. Then we insert them into the creds map in the order
	// in which they were supplied on the CLI.
	for _, file := range files {
		cset, err := credentials.Load(findCreds(credDir, file))
		if err != nil {
			return creds, err
		}
		res, err := cset.Resolve()
		if err != nil {
			return res, err
		}

		for k, v := range res {
			creds[k] = v
		}
	}
	return creds, credentials.Validate(creds, b.Credentials)
}

func findCreds(credDir string, file string) string {
	if !fileExists(file) {
		testPath := filepath.Join(credDir, file+".yaml")
		if fileExists(testPath) {
			file = testPath
		} else {
			file = filepath.Join(credDir, file+".yml") // Don't bother checking existence because it fails later
		}
	}
	return file
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func must(err error) {
	if err != nil {
		os.Exit(1)
	}
}

// prepareDriver prepares a driver per the user's request.
func prepareDriver(driverName string) (driver.Driver, error) {
	driverImpl, err := lookup.Lookup(driverName)
	if err != nil {
		return nil, err
	}

	if configurable, ok := driverImpl.(driver.Configurable); ok {
		configureDriver(configurable)
	}

	return driverImpl, nil
}

// configureDriver loads any driver-specific config out of the environment.
func configureDriver(configurable driver.Configurable) {
	driverCfg := map[string]string{}
	for env := range configurable.Config() {
		if val, ok := os.LookupEnv(env); ok {
			driverCfg[env] = val
		}
	}
	configurable.SetConfig(driverCfg)
}

func makeOpRelocator(relMapping string) (action.OperationConfigFunc, error) {
	rm, err := loadRelMapping(relMapping)
	if err != nil {
		return nil, err
	}

	relMap := make(map[string]string)
	if rm != "" {
		err := json.Unmarshal([]byte(rm), &relMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal relocation mapping: %v", err)
		}
	}

	return func(op *driver.Operation) error {
		// if there is a relocation mapping, ensure it is mounted and relocate the invocation image
		if rm != "" {
			op.Files["/cnab/app/relocation-mapping.json"] = rm

			im, ok := relMap[op.Image.Image]
			if !ok {
				return fmt.Errorf("invocation image %s not present in relocation mapping %v", op.Image.Image, relMap)
			}
			op.Image.Image = im
		}
		return nil
	}, nil
}

func loadRelMapping(relMap string) (string, error) {
	if relMap != "" {
		data, err := ioutil.ReadFile(relMap)
		if err != nil {
			return "", fmt.Errorf("failed to read relocation mapping from %s: %v", relMap, err)
		}
		return string(data), nil
	}

	return "", nil
}

// TODO
//
// remove from main
func loadBundle(bundleFile string) (*bundle.Bundle, error) {
	l := loader.NewLoader()

	// Issue #439: Errors that come back from the loader can be
	// pretty opaque.
	var bun *bundle.Bundle
	bun, err := l.Load(bundleFile)
	if err != nil {
		return bun, fmt.Errorf("cannot load bundle: %s", err)
	}
	return bun, nil
}
func getReference(bundleName string) (reference.NamedTagged, error) {
	var (
		name string
		ref  reference.NamedTagged
	)

	parts := strings.SplitN(bundleName, "://", 2)
	if len(parts) == 2 {
		name = parts[1]
	} else {
		name = parts[0]
	}
	normalizedRef, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid bundle name: %v", name, err)
	}
	if reference.IsNameOnly(normalizedRef) {
		ref, err = reference.WithTag(normalizedRef, "latest")
		if err != nil {
			// NOTE(bacongobbler): Using the default tag *must* be valid.
			// To create a NamedTagged type with non-validated
			// input, the WithTag function should be used instead.
			panic(err)
		}
	} else {
		if taggedRef, ok := normalizedRef.(reference.NamedTagged); ok {
			ref = taggedRef
		} else {
			return nil, fmt.Errorf("unsupported image name: %s", normalizedRef.String())
		}
	}

	return ref, nil
}
