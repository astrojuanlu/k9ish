// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsManifest(t *testing.T) {
	uu := map[string]struct {
		file string
		e    bool
	}{
		"yaml": {file: "fred.yaml", e: true},
		"yml":  {file: "fred.yml", e: true},
		"nope": {file: "fred.txt"},
	}

	for k := range uu {
		u := uu[k]
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, u.e, isManifest(u.file))
		})
	}
}
