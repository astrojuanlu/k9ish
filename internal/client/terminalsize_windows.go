// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

//go:build windows

package client

import (
	"os"

	"k8s.io/client-go/tools/remotecommand"
)

// terminalSizeQueue is a no-op stub: terminal resize notification is not
// supported on windows.
type terminalSizeQueue struct{}

func (terminalSizeQueue) Next() *remotecommand.TerminalSize { return nil }

func newTerminalSizeQueue(*os.File) remotecommand.TerminalSizeQueue {
	return terminalSizeQueue{}
}

func stopTerminalSizeQueue(remotecommand.TerminalSizeQueue) {}
