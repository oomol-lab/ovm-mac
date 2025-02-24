//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type cmd struct {
	// Path to command executable filename
	name string
	// Command args.
	args []string
	// SSH session.
	mySession *ssh.Session
	// Context for cancellation
	context context.Context
	// Signal send when the context is canceled
	signal ssh.Signal
}

// SetStopSignal sets the signal to send when the context is canceled.
func (c *cmd) SetStopSignal(signal ssh.Signal) {
	c.signal = signal
}

// String returns the command line string, with each parameter wrapped in ""
func (c *cmd) String() string {
	args := append([]string{c.name}, c.args...)
	for i, s := range args {
		args[i] = fmt.Sprintf("\"%s\"", s)
	}
	return strings.Join(args, " ")
}

// runCtx executes the given callback within session. Sends SIGINT when the context is canceled.
func (c *cmd) runCtx() error {
	outPipe, err := c.mySession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get session.StdoutPipe():%v", err)
	}
	errPipe, err := c.mySession.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get session.StderrPipe():%v", err)
	}

	logStdErr := func(pipe io.Reader, done chan struct{}) {
		_, err := io.Copy(os.Stderr, pipe)
		if err != nil {
			logrus.Errorf("failed to copy pipe into os.Stderr")
		}
		done <- struct{}{}
	}

	logStdOut := func(pipe io.Reader, done chan struct{}) {
		_, err := io.Copy(os.Stdout, pipe)
		if err != nil {
			logrus.Errorf("failed to copy pipe into os.Stdout")
		}
		done <- struct{}{}
	}

	// Send SIGINT when context is canceled by default
	if c.signal == "" {
		c.signal = ssh.SIGKILL
	}

	go func() {
		<-c.context.Done()
		// Send Kill to remote process for now, should we nicely send SIGINT?
		_ = c.mySession.Signal(c.signal)
		logrus.Warnf("Command [ %q ] was killed(SIGKILL), cause by %v", c.String(), context.Cause(c.context))
	}()

	if err := c.mySession.Start(c.String()); err != nil {
		return fmt.Errorf("failed to start ssh command:%v", err)
	}
	defer c.mySession.Close()

	completed := make(chan struct{}, 2) //nolint:mnd
	go logStdOut(outPipe, completed)
	go logStdErr(errPipe, completed)
	<-completed
	<-completed

	return c.mySession.Wait() //nolint:wrapcheck
}
