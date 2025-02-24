//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package ssh

import (
	"context"
	"fmt"

	"bauklotze/pkg/ssh"

	"github.com/sirupsen/logrus"
	sshSingal "golang.org/x/crypto/ssh"
)

func RunCtx(ctx context.Context, identityPath string, targetHost, username string, port uint, name string, args []string) error {
	// New auth key
	auth, err := ssh.NewAuthKey(identityPath)
	if err != nil {
		return fmt.Errorf("failed to get ssh key: %w", err)
	}

	// Make a new ssh client
	client, err := ssh.NewClient(targetHost, username, port, auth)
	if err != nil {
		return fmt.Errorf("failed to create ssh client: %w", err)
	}
	defer client.Close()

	// Create a new ssh command
	client, err = client.SetCmdLine(ctx, name, args)
	if err != nil {
		return fmt.Errorf("failed to create ssh command: %w", err)
	}

	// Set the signal to send when the context is canceled
	client.Cmd.SetStopSignal(sshSingal.SIGKILL)

	logrus.Infof("SSH client running command: %s", client.Cmd.String())
	return client.RunCtx() //nolint:wrapcheck
}

func Run(identityPath string, targetHost, username string, port uint, name string, args []string) error {
	ctx := context.Background()
	return RunCtx(ctx, identityPath, targetHost, username, port, name, args)
}
