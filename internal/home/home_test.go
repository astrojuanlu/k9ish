// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package home

import (
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
)

type userFunc func() (*user.User, error)

func withFakeUser(t *testing.T, fn userFunc) {
	t.Helper()
	orig := currentUser
	currentUser = fn
	t.Cleanup(func() { currentUser = orig })
}

func passwdUser(home string) userFunc {
	return func() (*user.User, error) { return &user.User{HomeDir: home}, nil }
}

func failingUser() userFunc {
	return func() (*user.User, error) { return nil, assert.AnError }
}

func TestDir(t *testing.T) {
	tests := []struct {
		name    string
		hint    string
		envHome string
		user    userFunc
		want    string
	}{
		{
			name:    "hint wins over everything",
			hint:    "/real/home",
			envHome: "/sandbox/home",
			user:    passwdUser("/real/home"),
			want:    "/real/home",
		},
		{
			name:    "account database wins over $HOME",
			envHome: "/sandbox/home",
			user:    passwdUser("/real/home"),
			want:    "/real/home",
		},
		{
			name:    "account lookup failure falls back to $HOME",
			envHome: "/plain/home",
			user:    failingUser(),
			want:    "/plain/home",
		},
		{
			name:    "empty account home falls back to $HOME",
			envHome: "/plain/home",
			user:    passwdUser(""),
			want:    "/plain/home",
		},
		{
			name: "no sources yields empty",
			user: failingUser(),
			want: "",
		},
		{
			name:    "plain environment agrees with $HOME",
			envHome: "/home/user",
			user:    passwdUser("/home/user"),
			want:    "/home/user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Control both sources explicitly: an empty value clears the
			// variable, simulating it being unset (execution environments
			// such as build sandboxes may export either of them globally).
			t.Setenv(realHomeEnvVar, tt.hint)
			t.Setenv("HOME", tt.envHome)
			withFakeUser(t, tt.user)

			assert.Equal(t, tt.want, Dir())
		})
	}
}

func TestDirEmptyHintIgnored(t *testing.T) {
	t.Setenv(realHomeEnvVar, "")
	withFakeUser(t, passwdUser("/real/home"))

	assert.Equal(t, "/real/home", Dir())
}
