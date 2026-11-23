// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellCommand(t *testing.T) {
	uu := map[string]struct {
		os string
		e  []string
	}{
		"linux": {
			os: "linux",
			e:  []string{"sh", "-c", shellCheck},
		},
		"darwin": {
			os: "darwin",
			e:  []string{"sh", "-c", shellCheck},
		},
		"windows": {
			os: windowsOS,
			e:  []string{"cmd", "/c", winShellCheck},
		},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, shellCommand(u.os))
		})
	}
}
