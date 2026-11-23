// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package client

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/derailed/k9s/internal/slogs"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/cmd/util"
)

// ExecSpec describes a pod exec or attach session.
type ExecSpec struct {
	// Namespace is the pod namespace.
	Namespace string
	// PodName is the pod name.
	PodName string
	// Container is the target container (blank for the default one).
	Container string
	// Command is the command to run (exec only).
	Command []string
	// TTY allocates a pseudo-terminal in the pod.
	TTY bool
	// Stdin streams the process stdin into the pod.
	Stdin bool
}

// PodExec runs a command inside a pod container, streaming the process
// terminal to and from the cluster.
func (a *APIClient) PodExec(ctx context.Context, spec *ExecSpec) error {
	return a.streamPod(ctx, spec, "exec")
}

// PodAttach attaches to a running pod container, streaming the process
// terminal to and from the cluster.
func (a *APIClient) PodAttach(ctx context.Context, spec *ExecSpec) error {
	return a.streamPod(ctx, spec, "attach")
}

func (a *APIClient) streamPod(ctx context.Context, spec *ExecSpec, verb string) error {
	if !a.ConnectionOK() {
		return fmt.Errorf("no connection to cluster")
	}
	dial, err := a.Dial()
	if err != nil {
		return err
	}
	cfg, err := a.RestConfig()
	if err != nil {
		return err
	}
	u, err := podStreamURL(dial, spec, verb)
	if err != nil {
		return err
	}
	exec, err := newStreamExecutor(cfg, u)
	if err != nil {
		return err
	}

	opts := remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: nil,
		Stderr: nil,
		Tty:    spec.TTY,
	}
	if spec.Stdin {
		opts.Stdin = os.Stdin
	}
	opts.Stdout = os.Stdout
	if !spec.TTY {
		opts.Stderr = os.Stderr
	}
	if spec.TTY {
		q := newTerminalSizeQueue(os.Stdout)
		opts.TerminalSizeQueue = q
		defer stopTerminalSizeQueue(q)
	}

	slog.Debug("Streaming pod session",
		slogs.Namespace, spec.Namespace,
		slogs.ResName, spec.PodName,
		slogs.Container, spec.Container,
		slogs.Command, spec.Command,
		slogs.Verb, verb,
	)

	return exec.StreamWithContext(ctx, opts)
}

// podStreamURL builds the exec/attach request URL for the given pod.
func podStreamURL(dial kubernetes.Interface, spec *ExecSpec, verb string) (*url.URL, error) {
	req := dial.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(spec.Namespace).
		Name(spec.PodName).
		SubResource(verb)

	switch verb {
	case "exec":
		req.VersionedParams(&v1.PodExecOptions{
			Container: spec.Container,
			Command:   spec.Command,
			Stdin:     spec.Stdin,
			Stdout:    true,
			Stderr:    !spec.TTY,
			TTY:       spec.TTY,
		}, scheme.ParameterCodec)
	case "attach":
		req.VersionedParams(&v1.PodAttachOptions{
			Container: spec.Container,
			Stdin:     spec.Stdin,
			Stdout:    true,
			Stderr:    !spec.TTY,
			TTY:       spec.TTY,
		}, scheme.ParameterCodec)
	default:
		return nil, fmt.Errorf("unsupported pod stream verb %q", verb)
	}

	return req.URL(), nil
}

// newStreamExecutor builds an executor with WebSocket as the primary protocol
// and SPDY as fallback, mirroring kubectl's behavior.
func newStreamExecutor(cfg *restclient.Config, u *url.URL) (remotecommand.Executor, error) {
	spd, err := remotecommand.NewSPDYExecutor(cfg, "POST", u)
	if err != nil {
		return nil, err
	}
	if util.RemoteCommandWebsockets.IsDisabled() {
		return spd, nil
	}
	// WebSocketExecutor must be "GET" method as described in RFC 6455 Sec. 4.1.
	ws, err := remotecommand.NewWebSocketExecutor(cfg, "GET", u.String())
	if err != nil {
		return nil, err
	}

	return remotecommand.NewFallbackExecutor(ws, spd, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
}
