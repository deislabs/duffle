package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pivotal/image-relocation/pkg/image"

	"github.com/deislabs/duffle/pkg/duffle/home"
	"github.com/deislabs/duffle/pkg/imagestore"
	"github.com/deislabs/duffle/pkg/imagestore/imagestoremocks"
)

const (
	testRepositoryPrefix = "example.com/user"

	originalInvocationImageName  = "technosophos/helloworld:0.1.0"
	relocatedInvocationImageName = "example.com/user/technosophos-helloworld-6731e0d41b7fd5a24e14c853af93bd81:0.1.0"

	originalImageNameA  = "deislabs/duffle@sha256:4d41eeb38fb14266b7c0461ef1ef0b2f8c05f41cd544987a259a9d92cdad2540"
	relocatedImageNameA = "example.com/user/deislabs-duffle-50aa5cc4ebb040ac696a9753d1695298@sha256:4d41eeb38fb14266b7c0461ef1ef0b2f8c05f41cd544987a259a9d92cdad2540"
	imageDigestA        = "sha256:4d41eeb38fb14266b7c0461ef1ef0b2f8c05f41cd544987a259a9d92cdad2540"

	originalImageNameB   = "deislabs/duffle:0.1.0-ralpha.5-englishrose"
	relocatedImageNameB  = "example.com/user/deislabs-duffle-50aa5cc4ebb040ac696a9753d1695298:0.1.0-ralpha.5-englishrose"
	originalImageDigestB = "sha256:14d6134d892aeccb7e142557fe746ccd0a8f736a747c195ef04c9f3f0f0bbd49"
)

func TestRelocateFileToFileSupportedImageTypes(t *testing.T) {
	relocateFileToFile(t, "testdata/relocate/bundle.json", nil, func(archiveDir string) {
		if archiveDir != "" {
			t.Fatalf("archiveDir was %q, expected %q", archiveDir, "")
		}
	})
}

func TestRelocateThickBundleToFileSupportedImageTypes(t *testing.T) {
	relocateFileToFile(t, "testdata/relocate/testrelocate-0.1.tgz", nil, func(archiveDir string) {
		expectedSuffix := "testrelocate-0.1"
		if !strings.HasSuffix(archiveDir, expectedSuffix) {
			t.Fatalf("archiveDir was %q, expected it to end with %q", archiveDir, expectedSuffix)
		}
	})
}

func TestRelocateFileToFileUnsupportedImageType(t *testing.T) {
	relocateFileToFile(t, "testdata/relocate/bundle-with-unsupported-image-type.json", errors.New("cannot relocate image c with imageType c: only oci and docker image types are currently supported"), func(archiveDir string) {
		if archiveDir != "" {
			t.Fatalf("archiveDir was %q, expected %q", archiveDir, "")
		}
	})
}

func relocateFileToFile(t *testing.T, bundle string, expectedErr error, archiveDirStub func(archiveDir string)) {
	duffleHome, err := ioutil.TempDir("", "dufflehome")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(duffleHome)

	work, err := ioutil.TempDir("", "relocatetest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(work)

	outputBundle := filepath.Join(work, "relocated.json")

	cmd := &relocateCmd{
		inputBundle:  bundle,
		outputBundle: outputBundle,

		repoPrefix:         testRepositoryPrefix,
		inputBundleIsFile:  true,
		outputBundleIsFile: true,

		home: home.Home(duffleHome),
		out:  ioutil.Discard,

		imageStoreBuilder: &imagestoremocks.MockBuilder{
			ArchiveDirStub: archiveDirStub,
			BuildStub: func() (imagestore.Store, error) {
				return &imagestoremocks.MockStore{
					PushStub: func(dig image.Digest, src image.Name, dst image.Name) error {
						type pair struct {
							first  string
							second string
						}
						digests := map[string]pair{
							"docker.io/" + originalInvocationImageName: {"", relocatedInvocationImageName},
							"docker.io/" + originalImageNameA:          {imageDigestA, relocatedImageNameA},
							"docker.io/" + originalImageNameB:          {originalImageDigestB, relocatedImageNameB},
						}
						exp, ok := digests[src.String()]
						if !ok {
							t.Fatalf("unexpected source image %v", src)
						}
						expectedDig, expectedDst := exp.first, exp.second
						if dig.String() != expectedDig {
							t.Fatalf("digest for source image %v was %s, expected %s", src, dig, expectedDig)
						}
						if dst.String() != expectedDst {
							t.Fatalf("destination image for source image %v was %v, expected %s", src, dst, expectedDst)
						}
						return nil
					},
				}, nil
			},
		},
	}

	if err := cmd.run(); (err != nil && expectedErr != nil && err.Error() != expectedErr.Error()) ||
		((err == nil || expectedErr == nil) && err != expectedErr) {
		t.Fatalf("unexpected error %v (expected %v)", err, expectedErr)
	}

	if expectedErr != nil {
		return
	}

	// check output bundle
	bundleFile, err := resolveBundleFilePath(outputBundle, "", true)
	if err != nil {
		t.Fatal(err)
	}

	bun, err := loadBundle(bundleFile)
	if err != nil {
		t.Fatal(err)
	}

	if err = bun.Validate(); err != nil {
		t.Fatal(err)
	}

	ii := bun.InvocationImages[0]
	actualInvocationImageName := ii.Image
	if actualInvocationImageName != relocatedInvocationImageName {
		t.Fatalf("output bundle has invocation image with unexpected name: %q (expected %q)",
			actualInvocationImageName, relocatedInvocationImageName)
	}

	actualOriginalInvocationImageName := ii.OriginalImage
	if actualOriginalInvocationImageName != originalInvocationImageName {
		t.Fatalf("output bundle has invocation image with unexpected original name: %q (expected %q)",
			actualOriginalInvocationImageName, originalInvocationImageName)
	}

	assertImage := func(i string, expectedOriginalImageName string, expectedImageName string) {
		img := bun.Images[i]

		actualImageName := img.Image
		if actualImageName != expectedImageName {
			t.Fatalf("output bundle has image %s with unexpected name: %q (expected %q)", i, actualImageName,
				expectedImageName)
		}

		actualOriginalImageName := img.OriginalImage
		if actualOriginalImageName != expectedOriginalImageName {
			t.Fatalf("output bundle has image %s with unexpected original name: %q (expected %q)", i,
				actualOriginalImageName, expectedOriginalImageName)
		}
	}

	assertImage("a", originalImageNameA, relocatedImageNameA)
	assertImage("b", originalImageNameB, relocatedImageNameB)
}
