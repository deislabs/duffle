package repo

import "strings"

// NamedRepositoryList is a list of bundle references.
// Implements a sorter on Name.
type NamedRepositoryList []*NamedRepository

// Len returns the length.
func (bl NamedRepositoryList) Len() int { return len(bl) }

// Swap swaps the position of two items in the versions slice.
func (bl NamedRepositoryList) Swap(i, j int) { bl[i], bl[j] = bl[j], bl[i] }

// Less returns true if the version of entry a is less than the version of entry b.
func (bl NamedRepositoryList) Less(a, b int) bool {
	return strings.Compare(bl[a].Name, bl[b].Name) < 1
}

// NamedRepository is a reference to a repository.
type NamedRepository struct {
	Name   string
	Tag    string
	Digest string
}

// Name returns the full name
func (n *NamedRepository) String() string {
	return n.Name + ":" + n.Tag
}
