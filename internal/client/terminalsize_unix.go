// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

//go:build !windows

package client

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
	"k8s.io/client-go/tools/remotecommand"
)

// terminalSizeQueue reports terminal resize events to the stream executor.
type terminalSizeQueue struct {
	fd    int
	sizes chan remotecommand.TerminalSize
	done  chan struct{}
	once  sync.Once
}

// newTerminalSizeQueue monitors the given terminal file for resize events.
func newTerminalSizeQueue(f *os.File) remotecommand.TerminalSizeQueue {
	q := &terminalSizeQueue{
		fd:    int(f.Fd()),
		sizes: make(chan remotecommand.TerminalSize, 1),
		done:  make(chan struct{}),
	}
	go q.watch()
	q.push()

	return q
}

func (q *terminalSizeQueue) push() {
	w, h, err := term.GetSize(q.fd)
	if err != nil {
		return
	}
	select {
	case q.sizes <- remotecommand.TerminalSize{Width: uint16(w), Height: uint16(h)}:
	default:
	}
}

func (q *terminalSizeQueue) watch() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGWINCH)
	defer signal.Stop(sig)

	for {
		select {
		case <-q.done:
			return
		case <-sig:
			q.push()
		}
	}
}

// Next returns the new terminal size after a resize, or nil once stopped.
func (q *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	select {
	case sz := <-q.sizes:
		return &sz
	case <-q.done:
		return nil
	}
}

func (q *terminalSizeQueue) stop() {
	q.once.Do(func() { close(q.done) })
}

func stopTerminalSizeQueue(q remotecommand.TerminalSizeQueue) {
	if s, ok := q.(interface{ stop() }); ok {
		s.stop()
	}
}
