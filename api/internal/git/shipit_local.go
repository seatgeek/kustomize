// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const (
	shipItRepoPath       = "platform/ship-it.git"
	gitlabHostnameEnvVar = "GITLAB_HOSTNAME"
)

var (
	shipItRootOnce     sync.Once
	shipItRoot         string
	shipItRootOk       bool
	shipItURLRegexOnce sync.Once
	shipItURLRegex     *regexp.Regexp
)

func normalizeGitlabHostname(raw string) string {
	host := strings.TrimSpace(raw)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	return strings.TrimSuffix(host, "/")
}

// gitlabHostname returns the GitLab host from GITLAB_HOSTNAME.
func gitlabHostname() string {
	return normalizeGitlabHostname(os.Getenv(gitlabHostnameEnvVar))
}

func isShipItRepoSpec(rs *RepoSpec) bool {
	if rs == nil || rs.RepoPath != shipItRepoPath {
		return false
	}
	host := gitlabHostname()
	if host == "" {
		return false
	}
	return strings.Contains(rs.Host, host)
}

// ShipItURLRegex matches remote ship-it kustomize bases for the configured GitLab host.
// Returns nil when GITLAB_HOSTNAME is unset.
func ShipItURLRegex() *regexp.Regexp {
	shipItURLRegexOnce.Do(func() {
		host := gitlabHostname()
		if host == "" {
			return
		}
		pattern := fmt.Sprintf(
			`https://%s/platform/ship-it\.git//([^?]+)\?ref=\S+`,
			regexp.QuoteMeta(host),
		)
		shipItURLRegex = regexp.MustCompile(pattern)
	})
	return shipItURLRegex
}

// GitTopLevel returns the git repository root for dir, or "" if unavailable.
func GitTopLevel(dir string) string {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func dirHasKustomize(path string) bool {
	info, err := os.Stat(filepath.Join(path, "kustomize"))
	return err == nil && info.IsDir()
}

func isShipItCheckout(path string) bool {
	if !dirHasKustomize(path) {
		return false
	}
	out, err := exec.Command("git", "-C", path, "remote", "get-url", "origin").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "platform/ship-it")
}

// DiscoverShipItRoot finds a local platform/ship-it checkout near anchorDir.
func DiscoverShipItRoot(anchorDir string) (string, bool) {
	shipItRootOnce.Do(func() {
		start := anchorDir
		if start == "" {
			start, _ = os.Getwd()
		}
		if abs, err := filepath.Abs(start); err == nil {
			start = abs
		}
		if gitRoot := GitTopLevel(start); gitRoot != "" {
			if isShipItCheckout(gitRoot) {
				shipItRoot = gitRoot
				shipItRootOk = true
				return
			}
			start = gitRoot
		}
		dir := start
		for {
			candidate := filepath.Join(dir, "platform", "ship-it")
			if isShipItCheckout(candidate) {
				shipItRoot = candidate
				shipItRootOk = true
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	})
	return shipItRoot, shipItRootOk
}

// RelPathFromShipItURL rewrites a remote ship-it git base to a path relative to loaderRoot.
// Used when fast workspace preparation is not applicable (CI fallback).
func RelPathFromShipItURL(rawURL string, loaderRoot string) (string, bool) {
	rs, err := NewRepoSpecFromURL(rawURL)
	if err != nil || !isShipItRepoSpec(rs) {
		return "", false
	}
	shipItRoot, ok := DiscoverShipItRoot(loaderRoot)
	if !ok {
		return "", false
	}
	absPath := filepath.Clean(filepath.Join(shipItRoot, rs.KustRootPath))
	rel, err := filepath.Rel(loaderRoot, absPath)
	if err != nil {
		return "", false
	}
	resolved := filepath.Clean(filepath.Join(loaderRoot, rel))
	if resolved != absPath {
		return "", false
	}
	return rel, true
}

// TryLocalShipIt binds rs.Dir to a local platform/ship-it checkout when discoverable.
func TryLocalShipIt(rs *RepoSpec, anchorDir string) (bool, error) {
	if !isShipItRepoSpec(rs) {
		return false, nil
	}
	root, ok := DiscoverShipItRoot(anchorDir)
	if !ok {
		return false, nil
	}
	confirmed, err := filesys.ConfirmDir(filesys.MakeFsOnDisk(), root)
	if err != nil {
		return false, nil
	}
	rs.Dir = confirmed
	return true, nil
}
