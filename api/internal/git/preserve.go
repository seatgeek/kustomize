// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// ShouldPreserveCloneDir reports whether a git clone directory must not be removed on loader cleanup.
// Cached clones are shared across loaders; local ship-it checkouts are never temp clones.
func ShouldPreserveCloneDir(dir filesys.ConfirmedDir) bool {
	if dir == "" || dir == notCloned {
		return false
	}
	if IsCachedCloneDir(dir) {
		return true
	}
	if _, ok := DiscoverShipItRoot(dir.String()); ok {
		return true
	}
	return false
}

// SafeCleaner returns a cleanup func that skips RemoveAll for preserved clone dirs.
func SafeCleaner(repoSpec *RepoSpec, fSys filesys.FileSystem) func() error {
	remove := repoSpec.Cleaner(fSys)
	return func() error {
		if ShouldPreserveCloneDir(repoSpec.Dir) {
			return nil
		}
		return remove()
	}
}
