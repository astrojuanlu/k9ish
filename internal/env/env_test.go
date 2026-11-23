// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Run("primary wins", func(t *testing.T) {
		require.NoError(t, os.Setenv("TEST_ENV_PRIMARY", "a"))
		defer os.Unsetenv("TEST_ENV_PRIMARY")
		require.NoError(t, os.Setenv("TEST_ENV_FALLBACK", "b"))
		defer os.Unsetenv("TEST_ENV_FALLBACK")

		assert.Equal(t, "a", Get("TEST_ENV_PRIMARY", "TEST_ENV_FALLBACK"))
	})

	t.Run("fallback used when primary unset", func(t *testing.T) {
		require.NoError(t, os.Setenv("TEST_ENV_FALLBACK", "b"))
		defer os.Unsetenv("TEST_ENV_FALLBACK")

		assert.Equal(t, "b", Get("TEST_ENV_PRIMARY", "TEST_ENV_FALLBACK"))
	})

	t.Run("unset", func(t *testing.T) {
		assert.Empty(t, Get("TEST_ENV_PRIMARY", "TEST_ENV_FALLBACK"))
	})

	t.Run("empty value counts as unset", func(t *testing.T) {
		require.NoError(t, os.Setenv("TEST_ENV_PRIMARY", ""))
		defer os.Unsetenv("TEST_ENV_PRIMARY")
		require.NoError(t, os.Setenv("TEST_ENV_FALLBACK", "b"))
		defer os.Unsetenv("TEST_ENV_FALLBACK")

		assert.Equal(t, "b", Get("TEST_ENV_PRIMARY", "TEST_ENV_FALLBACK"))
	})
}

func TestIsSet(t *testing.T) {
	defer os.Unsetenv("TEST_ENV_PRIMARY")
	defer os.Unsetenv("TEST_ENV_FALLBACK")

	assert.False(t, IsSet("TEST_ENV_PRIMARY", "TEST_ENV_FALLBACK"))

	require.NoError(t, os.Setenv("TEST_ENV_FALLBACK", "b"))
	assert.True(t, IsSet("TEST_ENV_PRIMARY", "TEST_ENV_FALLBACK"))
}
