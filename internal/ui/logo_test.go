// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package ui_test

import (
	"testing"

	"github.com/derailed/k9s/internal/config"
	"github.com/derailed/k9s/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestNewLogoView(t *testing.T) {
	v := ui.NewLogo(config.NewStyles())
	v.Reset()

	const elogo = "[#ffa500::b] ____  __ ________\n[#ffa500::b]|    |/  /   __   \\\n[#ffa500::b]|       /\\____    /\n[#ffa500::b]|    \\   \\  /    /\n[#ffa500::b]|____|\\__ \\/____/\n[#ffa500::b]         \\/ish\n"
	assert.Equal(t, elogo, v.Logo().GetText(false))
	assert.Empty(t, v.Status().GetText(false))
}

func TestLogoStatus(t *testing.T) {
	uu := map[string]struct {
		logo, msg, e string
	}{
		"info": {
			"[#008000::b] ____  __ ________\n[#008000::b]|    |/  /   __   \\\n[#008000::b]|       /\\____    /\n[#008000::b]|    \\   \\  /    /\n[#008000::b]|____|\\__ \\/____/\n[#008000::b]         \\/ish\n",
			"blee",
			"[#ffffff::b]blee\n",
		},
		"warn": {
			"[#c71585::b] ____  __ ________\n[#c71585::b]|    |/  /   __   \\\n[#c71585::b]|       /\\____    /\n[#c71585::b]|    \\   \\  /    /\n[#c71585::b]|____|\\__ \\/____/\n[#c71585::b]         \\/ish\n",
			"blee",
			"[#ffffff::b]blee\n",
		},
		"err": {
			"[#ff0000::b] ____  __ ________\n[#ff0000::b]|    |/  /   __   \\\n[#ff0000::b]|       /\\____    /\n[#ff0000::b]|    \\   \\  /    /\n[#ff0000::b]|____|\\__ \\/____/\n[#ff0000::b]         \\/ish\n",
			"blee",
			"[#ffffff::b]blee\n",
		},
	}

	v := ui.NewLogo(config.NewStyles())
	for n := range uu {
		k, u := n, uu[n]
		t.Run(k, func(t *testing.T) {
			switch k {
			case "info":
				v.Info(u.msg)
			case "warn":
				v.Warn(u.msg)
			case "err":
				v.Err(u.msg)
			}
			assert.Equal(t, u.logo, v.Logo().GetText(false))
			assert.Equal(t, u.e, v.Status().GetText(false))
		})
	}
}
