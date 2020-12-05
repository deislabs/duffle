package duffle

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/cnabio/duffle/pkg/duffle/home"
	"github.com/cnabio/duffle/pkg/reference"
	"github.com/cnabio/duffle/pkg/repo"
)

func getBundleFilePath(name string, home home.Home) (string, error) {
	ref, err := getReference(name)
	if err != nil {
		return "", fmt.Errorf("could not parse reference for %s: %v", name, err)
	}

	// read the bundle reference from repositories.json
	index, err := repo.LoadIndex(home.Repositories())
	if err != nil {
		return "", fmt.Errorf("cannot open %s: %v", home.Repositories(), err)
	}

	tag := ref.Tag()
	if ref.Tag() == "latest" {
		tag = ""
	}

	digest, err := index.Get(ref.Name(), tag)
	if err != nil {
		return "", fmt.Errorf("could not find %s:%s in %s: %v", ref.Name(), ref.Tag(), home.Repositories(), err)
	}
	return filepath.Join(home.Bundles(), digest), nil
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

// deleteBundleVersions removes the given SHAs from bundle storage
//
// It warns, but does not fail, if a given SHA is not found.
func deleteBundleVersions(w io.Writer, vers []repo.BundleVersion, index repo.Index, h home.Home) {
	for _, ver := range vers {
		fpath := filepath.Join(h.Bundles(), ver.Digest)
		if err := os.Remove(fpath); err != nil {
			fmt.Fprintf(w, "WARNING: could not delete stake record %q", fpath)
		}
	}
}

func removeVersions(w io.Writer, home home.Home, index repo.Index, name, vn string, vers []repo.BundleVersion) error {
	matcher, err := semver.NewConstraint(vn)
	if err != nil {
		return err
	}

	deletions := []repo.BundleVersion{}
	for _, ver := range vers {
		if ok, _ := matcher.Validate(ver.Version); ok {
			fmt.Fprintf(w, "Version %s matches constraint %q\n", ver, vn)
			deletions = append(deletions, ver)
			index.DeleteVersion(name, ver.Version.String())
			// If there are no more versions, remove the entire entry.
			if vers, ok := index.GetVersions(name); ok && len(vers) == 0 {
				index.Delete(name)
			}
		}
	}

	if len(deletions) == 0 {
		return nil
	}
	if err := index.WriteFile(home.Repositories(), 0644); err != nil {
		return err
	}
	deleteBundleVersions(w, deletions, index, home)
	return nil
}
