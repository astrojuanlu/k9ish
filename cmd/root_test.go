// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestSeedKubeConfigFrom(t *testing.T) {
	t.Run("explicit flag wins", func(t *testing.T) {
		flags := genericclioptions.NewConfigFlags(false)
		flags.KubeConfig = ptr("/explicit/config.yaml")

		seedKubeConfigFrom(flags, "/env/config.yaml", "/real/home")

		require.NotNil(t, flags.KubeConfig)
		assert.Equal(t, "/explicit/config.yaml", *flags.KubeConfig)
	})

	t.Run("KUBECONFIG env wins", func(t *testing.T) {
		flags := genericclioptions.NewConfigFlags(false)

		seedKubeConfigFrom(flags, "/env/config.yaml", "/real/home")

		require.NotNil(t, flags.KubeConfig)
		assert.Empty(t, *flags.KubeConfig)
	})

	t.Run("seeds real home kubeconfig", func(t *testing.T) {
		flags := genericclioptions.NewConfigFlags(false)

		seedKubeConfigFrom(flags, "", "/real/home")

		require.NotNil(t, flags.KubeConfig)
		assert.Equal(t, "/real/home/.kube/config", *flags.KubeConfig)
	})

	t.Run("skips when home is unknown", func(t *testing.T) {
		flags := genericclioptions.NewConfigFlags(false)

		seedKubeConfigFrom(flags, "", "")

		require.NotNil(t, flags.KubeConfig)
		assert.Empty(t, *flags.KubeConfig)
	})
}

func ptr(s string) *string { return &s }
