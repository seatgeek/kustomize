// Copyright 2022 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package localizer

import (
	"path/filepath"

	"sigs.k8s.io/kustomize/api/internal/git"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const defaultShipItRef = "master"

// seedLocalShipItKustomize copies the local platform/ship-it kustomize tree into dst under
// localized-files/{host}/platform/ship-it/{ref}/kustomize so localize can materialize
// remote ship-it references inside the output tree instead of symlinking outside scope.
func seedLocalShipItKustomize(fSys filesys.FileSystem, anchor, dst string) error {
	shipIt, ok := git.LocalShipItRoot(anchor)
	if !ok {
		return nil
	}
	host := git.GitlabHostname()
	if host == "" {
		return nil
	}
	seedDst := filepath.Join(dst, LocalizeDir, host, "platform", "ship-it", defaultShipItRef, "kustomize")
	if fSys.Exists(seedDst) {
		return nil
	}
	src := shipIt.Join("kustomize")
	if !fSys.Exists(src) {
		return nil
	}
	lc := &localizer{fSys: fSys}
	if err := lc.copyDir(filesys.ConfirmedDir(src), seedDst); err != nil {
		return errors.WrapPrefixf(err, "unable to seed local ship-it into %q", seedDst)
	}
	return nil
}
