// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

func testClientset(t *testing.T) kubernetes.Interface {
	t.Helper()
	clt, err := kubernetes.NewForConfig(&restclient.Config{Host: "https://localhost:6443"})
	require.NoError(t, err)

	return clt
}

func TestPodStreamURLExec(t *testing.T) {
	u, err := podStreamURL(testClientset(t), &ExecSpec{
		Namespace: "ns1",
		PodName:   "p1",
		Container: "c1",
		Command:   []string{"sh", "-c", "echo hi"},
		TTY:       true,
		Stdin:     true,
	}, "exec")
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/namespaces/ns1/pods/p1/exec", u.Path)
	q := u.Query()
	assert.Equal(t, "c1", q.Get("container"))
	assert.Equal(t, []string{"sh", "-c", "echo hi"}, q["command"])
	assert.Equal(t, "true", q.Get("stdin"))
	assert.Equal(t, "true", q.Get("stdout"))
	// False bools are omitted from the query string (omitempty).
	assert.Empty(t, q.Get("stderr"))
	assert.Equal(t, "true", q.Get("tty"))
}

func TestPodStreamURLExecNoTTY(t *testing.T) {
	u, err := podStreamURL(testClientset(t), &ExecSpec{
		Namespace: "ns1",
		PodName:   "p1",
		Stdin:     true,
	}, "exec")
	require.NoError(t, err)

	q := u.Query()
	assert.Empty(t, q.Get("container"))
	assert.Empty(t, q["command"])
	assert.Equal(t, "true", q.Get("stderr"))
	assert.Empty(t, q.Get("tty"))
}

func TestPodStreamURLAttach(t *testing.T) {
	u, err := podStreamURL(testClientset(t), &ExecSpec{
		Namespace: "ns1",
		PodName:   "p1",
		Container: "c1",
		Command:   []string{"ignored"},
		TTY:       true,
		Stdin:     true,
	}, "attach")
	require.NoError(t, err)

	assert.Equal(t, "/api/v1/namespaces/ns1/pods/p1/attach", u.Path)
	q := u.Query()
	assert.Equal(t, "c1", q.Get("container"))
	assert.Empty(t, q["command"])
	assert.Equal(t, "true", q.Get("stdin"))
	assert.Equal(t, "true", q.Get("tty"))
}

func TestPodStreamURLBadVerb(t *testing.T) {
	_, err := podStreamURL(testClientset(t), &ExecSpec{Namespace: "ns1", PodName: "p1"}, "evacuate")

	assert.Error(t, err)
}
