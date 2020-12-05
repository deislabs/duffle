package duffle

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/loader"
	"github.com/cnabio/duffle/pkg/duffle/home"
	"github.com/cnabio/duffle/pkg/repo"

	"github.com/gosuri/uitable"
)

// Show writes details of a bundle
func Show(w io.Writer, home home.Home, name string, raw bool) error {
	b, err := LoadFromReference(name, home)
	if err != nil {
		return err
	}

	if raw {
		_, err = b.WriteTo(w)
		return nil
	}

	d, err := json.MarshalIndent(b, " ", " ")
	if err != nil {
		return err
	}
	_, err = w.Write(d)

	return nil
}

// Remove deletes one or more bundles and their reference from the local store
func Remove(w io.Writer, name string, home home.Home, versions string) error {
	index, err := repo.LoadIndex(home.Repositories())
	if err != nil {
		return err
	}

	vers, ok := index.GetVersions(name)
	if !ok {
		fmt.Fprintf(w, "Bundle %q not found. Nothing deleted.", name)
		return nil
	}

	// if versions is set, we short circuit and only delete specific versions.
	if versions != "" {
		err = removeVersions(w, home, index, name, versions, vers)
		if err != nil {
			return err
		}
	}

	// if no version was specified, delete entire record
	if !index.Delete(name) {
		fmt.Fprintf(w, "Bundle %q not found. Nothing deleted.", name)
		return nil
	}
	if err := index.WriteFile(home.Repositories(), 0644); err != nil {
		return err
	}

	deleteBundleVersions(w, vers, index, home)
	return nil
}

// LoadFromReference returns a bundle given a local reference
func LoadFromReference(name string, home home.Home) (*bundle.Bundle, error) {
	f, err := getBundleFilePath(name, home)
	if err != nil {
		return nil, err
	}

	return Load(f)
}

// Load loads a bundle from a given file
func Load(bundleFile string) (*bundle.Bundle, error) {
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

// SearchLocal searches for a bundle in the local bundle store
func SearchLocal(home home.Home) (repo.NamedRepositoryList, error) {
	references := repo.NamedRepositoryList{}

	index, err := repo.LoadIndex(home.Repositories())
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %v", home.Repositories(), err)
	}

	for r, tagList := range index {
		for tag, digest := range tagList {
			_, err := Load(filepath.Join(home.Bundles(), digest))
			if err != nil {
				return nil, err
			}
			references = append(references, &repo.NamedRepository{
				Name:   r,
				Tag:    tag,
				Digest: digest,
			})
		}
	}

	sort.Sort(references)
	return references, nil
}

// List writes all bundles from a give Duffle home
func List(w io.Writer, home home.Home, short bool) error {
	ref, err := SearchLocal(home)
	if err != nil {
		return fmt.Errorf("cannot search local bundles: %v", err)
	}

	if short {
		for _, r := range ref {
			fmt.Fprintln(w, r.Name)
		}
		return nil
	}

	table := uitable.New()
	table.AddRow("NAME", "VERSION", "DIGEST")
	for _, r := range ref {
		table.AddRow(r.Name, r.Tag, r.Digest)
	}
	fmt.Fprintln(w, table)

	return nil
}
