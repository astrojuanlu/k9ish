// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

// Package home resolves the user's real home directory.
//
// Execution environments sometimes override $HOME to point at private
// per-application storage (e.g. sandboxed package managers, CI runners).
// Relying on $HOME alone would then miss the user's actual files. Dir()
// consults, in order:
//
//  1. an optional real-home hint exported by the execution environment,
//  2. the OS account database,
//  3. the $HOME environment variable.
//
// In unconfined environments all three sources agree, so the result matches
// what a plain $HOME lookup would have produced.
package home

import (
	"log/slog"
	"os"
	"os/user"

	"github.com/derailed/k9s/internal/slogs"
)

// realHomeEnvVar is an optional environment hint naming the user's real home
// directory. Environments that remap $HOME to private storage may set it so
// applications can still locate the user's actual home directory.
const realHomeEnvVar = "SNAP_REAL_HOME"

// currentUser is injectable for testing.
var currentUser = user.Current

// Dir returns the user's real home directory or an empty string if it cannot
// be determined.
func Dir() string {
	if h := os.Getenv(realHomeEnvVar); h != "" {
		return h
	}
	if usr, err := currentUser(); err == nil && usr != nil && usr.HomeDir != "" {
		return usr.HomeDir
	}
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	slog.Warn("Unable to determine the user home directory", slogs.Error, "all sources exhausted")

	return ""
}

// MustDir returns the user's real home directory or panics if it cannot be
// determined.
func MustDir() string {
	h := Dir()
	if h == "" {
		slog.Error("Die getting user home directory")
		os.Exit(1)
	}

	return h
}
