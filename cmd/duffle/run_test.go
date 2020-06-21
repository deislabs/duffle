package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/stretchr/testify/assert"
)

func TestRunCustom_ClaimOnly(t *testing.T) {
	claim, err := claim.New("some-claim")
	assert.NoError(t, err)

	claim.Bundle = &bundle.Bundle{
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					ImageType: "docker",
					Image:     "some-image",
				},
			},
		},
		Name: "some-bundle",
		Actions: map[string]bundle.Action{
			"some-custom-action": {
				Modifies:  false,
				Stateless: true,
			},
		},
	}

	claimStore := mockClaimStore()
	err = claimStore.Store(*claim)
	assert.NoError(t, err)

	buf := bytes.NewBuffer([]byte{})

	run := &runCmd{
		claimName: "some-claim",
		action:    "some-custom-action",
		out:       buf,
		driver:    "debug",
		storage:   &claimStore,
	}

	run.opOutFunc = setOut(run.out)

	err = run.run()
	assert.NoError(t, err)

	data := buf.String()
	if len(data) == 0 || !strings.Contains(data, "some-claim") {
		t.Fatalf("Expected driver to have received claim information, recieved: %s", data)
	}
}

func TestRunCustom_BundleName(t *testing.T) {
	tempDuffleHome, err := setupTempDuffleHome(t)
	assert.NoError(t, err)

	defer os.Remove(tempDuffleHome)

	err = copyTestBundle(tempDuffleHome)
	assert.NoError(t, err)

	claim, err := claim.New("foo-claim")
	assert.NoError(t, err)

	claimStore := mockClaimStore()
	err = claimStore.Store(*claim)
	assert.NoError(t, err)

	buf := bytes.NewBuffer([]byte{})

	run := &runCmd{
		bundleName: "foo",
		claimName:  "foo-claim",
		action:     "foo-action",
		out:        buf,
		driver:     "debug",
		storage:    &claimStore,
		home:       tempDuffleHome,
	}

	run.opOutFunc = setOut(run.out)

	err = run.run()
	assert.NoError(t, err)

	data := buf.String()
	if len(data) == 0 || !strings.Contains(data, "foo-action") {
		t.Fatalf("Expected driver to have received claim information, recieved: %s", data)
	}
}

func TestRunCustom_BundlePath(t *testing.T) {
	fooBundlePath := filepath.Join("..", "..", "tests", "testdata", "bundles", "foo.json")

	buf := bytes.NewBuffer([]byte{})

	run := &runCmd{
		bundlePath: fooBundlePath,
		claimName:  "foo-bundle",
		action:     "bar-action",
		out:        buf,
		driver:     "debug",
	}

	run.opOutFunc = setOut(run.out)

	err := run.run()
	assert.NoError(t, err)

	data := buf.String()
	if len(data) == 0 || !strings.Contains(data, "bar") {
		t.Fatalf("Expected driver to have received claim information, recieved: %s", data)
	}
}

func TestPrepareClaim_BundleErrorCases(t *testing.T) {
	// error when both bundle path and name are specified
	run := &runCmd{
		bundleName: "some-bundle-name",
		bundlePath: "some-bundle-path",
	}
	err := run.prepareClaim()
	assert.EqualError(t, err, `cannot specify both --bundle and --bundle-is-file: received bundle "some-bundle-name" and bundle file "some-bundle-path"`)

	// error when the bundle name does not exist in the duffle store
	run = &runCmd{bundleName: "some-unknown-bundle"}
	err = run.prepareClaim()
	assert.EqualError(t, err, `could not find some-unknown-bundle:latest in repositories.json: no bundle name found`)

	// error when the bundle path does not exist
	run = &runCmd{bundlePath: "non-existent-path"}
	err = run.prepareClaim()
	assert.EqualError(t, err, `bundle file "non-existent-path" does not exist`)
}
