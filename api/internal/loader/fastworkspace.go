// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/api/internal/git"
)

const manifestsDir = "_infrastructure/manifests"

// tryPrepareFastWorkspace mirrors letsgo platform kustomize fastbuild: copy app manifests,
// symlink local ship-it, rewrite remote bases in kustomization files to local paths.
func tryPrepareFastWorkspace(target string) (string, func() error, bool) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", nil, false
	}
	gitRoot := git.GitTopLevel(absTarget)
	if gitRoot == "" {
		return "", nil, false
	}
	shipIt, ok := git.DiscoverShipItRoot(absTarget)
	if !ok {
		return "", nil, false
	}
	manifestsRoot := filepath.Join(gitRoot, manifestsDir)
	relTarget, err := filepath.Rel(manifestsRoot, absTarget)
	if err != nil || strings.HasPrefix(relTarget, "..") {
		return "", nil, false
	}
	tmpdir, err := os.MkdirTemp("", "kustomize-fast-*")
	if err != nil {
		return "", nil, false
	}
	cleanup := func() error { return os.RemoveAll(tmpdir) }
	manifestsDst := filepath.Join(tmpdir, "manifests")
	if err := copyDir(manifestsRoot, manifestsDst); err != nil {
		_ = cleanup()
		return "", nil, false
	}
	shipItLink := filepath.Join(tmpdir, "ship-it")
	if err := os.Symlink(shipIt, shipItLink); err != nil {
		_ = cleanup()
		return "", nil, false
	}
	if err := rewriteShipItURLs(manifestsDst, tmpdir); err != nil {
		_ = cleanup()
		return "", nil, false
	}
	newTarget := filepath.Join(manifestsDst, relTarget)
	if _, err := os.Stat(newTarget); err != nil {
		_ = cleanup()
		return "", nil, false
	}
	return newTarget, cleanup, true
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, srcPath)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

func rewriteShipItURLs(manifestsDir, tmpdir string) error {
	return filepath.Walk(manifestsDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		baseName := filepath.Base(filePath)
		if baseName != "kustomization.yaml" && baseName != "kustomization.yml" {
			return nil
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		fileDir := filepath.Dir(filePath)
		relToShipIt, err := filepath.Rel(fileDir, filepath.Join(tmpdir, "ship-it"))
		if err != nil {
			return err
		}
		shipItURLRegex := git.ShipItURLRegex()
		if shipItURLRegex == nil {
			return nil
		}
		rewritten := shipItURLRegex.ReplaceAllStringFunc(string(data), func(match string) string {
			submatches := shipItURLRegex.FindStringSubmatch(match)
			if len(submatches) < 2 {
				return match
			}
			return filepath.Join(relToShipIt, submatches[1])
		})
		if rewritten != string(data) {
			return os.WriteFile(filePath, []byte(rewritten), info.Mode())
		}
		return nil
	})
}
