// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

// Package env provides environment variable lookup with fallback aliases.
//
// Application settings may be expressed either through current variable
// names or legacy aliases. The helpers below resolve the first non-empty
// value among the given names, which keeps renames backward-compatible
// without any packaging-specific logic.
package env

import "os"

// Get returns the value of the first environment variable whose value is
// non-empty, checking name first and then any fallback aliases. It returns
// "" when none are set.
func Get(name string, fallbacks ...string) string {
	for _, n := range append([]string{name}, fallbacks...) {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}

	return ""
}

// IsSet reports whether the given environment variable, or any of its
// fallback aliases, is set to a non-empty value.
func IsSet(name string, fallbacks ...string) bool {
	return Get(name, fallbacks...) != ""
}
