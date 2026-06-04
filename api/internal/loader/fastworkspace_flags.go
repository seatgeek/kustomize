// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package loader

// fastWorkspaceDisabled disables tryPrepareFastWorkspace (e.g. for kustomize localize).
var fastWorkspaceDisabled bool

// DisableFastWorkspace turns off the fast workspace optimization until EnableFastWorkspace.
func DisableFastWorkspace() {
	fastWorkspaceDisabled = true
}

// EnableFastWorkspace re-enables the fast workspace optimization.
func EnableFastWorkspace() {
	fastWorkspaceDisabled = false
}
