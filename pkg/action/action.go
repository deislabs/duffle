package action

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/scothis/ruffle/pkg/bundle"
	"github.com/scothis/ruffle/pkg/claim"
	"github.com/scothis/ruffle/pkg/credentials"
	"github.com/scothis/ruffle/pkg/driver"
)

// Action describes one of the primary actions that can be executed in CNAB.
//
// The actions are:
// - install
// - upgrade
// - uninstall
// - downgrade
// - status
type Action interface {
	// Run an action, and record the status in the given claim
	Run(*claim.Claim, credentials.Set) error
}

func selectInvocationImage(d driver.Driver, c *claim.Claim) (bundle.InvocationImage, error) {
	if len(c.Bundle.InvocationImages) == 0 {
		return bundle.InvocationImage{}, errors.New("no invocationImages are defined in the bundle")
	}

	for _, ii := range c.Bundle.InvocationImages {
		if d.Handles(ii.ImageType) {
			return ii, nil
		}
	}

	return bundle.InvocationImage{}, errors.New("driver is not compatible with any of the invocation images in the bundle")
}

func opFromClaim(action string, c *claim.Claim, ii bundle.InvocationImage, creds credentials.Set, w io.Writer) (*driver.Operation, error) {
	env, files, err := creds.Expand(c.Bundle)

	// Quick verification that no params were passed that are not actual legit params.
	for key := range c.Parameters {
		if _, ok := c.Bundle.Parameters[key]; !ok {
			return nil, fmt.Errorf("undefined parameter %q", key)
		}
	}

	for k, param := range c.Bundle.Parameters {
		rawval, ok := c.Parameters[k]
		if !ok {
			continue
		}
		value := fmt.Sprintf("%v", rawval)
		if param.Destination == nil {
			// env is a CNAB_P_
			env[fmt.Sprintf("CNAB_P_%s", strings.ToUpper(k))] = value
			continue
		}
		if param.Destination.Path != "" {
			files[param.Destination.Path] = value
		}
		if param.Destination.EnvironmentVariable != "" {
			env[param.Destination.EnvironmentVariable] = value
		}
	}

	env["CNAB_INSTALLATION_NAME"] = c.Name
	env["CNAB_ACTION"] = action
	env["CNAB_BUNDLE_NAME"] = c.Bundle.Name
	env["CNAB_BUNDLE_VERSION"] = c.Bundle.Version

	return &driver.Operation{
		Action:       action,
		Installation: c.Name,
		Parameters:   c.Parameters,
		Image:        ii.Image,
		ImageType:    ii.ImageType,
		Revision:     c.Revision,
		Environment:  env,
		Files:        files,
		Out:          w,
	}, err
}
