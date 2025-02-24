//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package ssh

import (
	"context"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	sshClient *ssh.Client
	Cmd       *cmd
}

const tcpProto = "tcp"

// NewClient New starts a new ssh connection, the host public key must be in known hosts.
func NewClient(addr, user string, port uint, auth []ssh.AuthMethod) (*Client, error) {
	sshClient, err := ssh.Dial(
		tcpProto,
		net.JoinHostPort(addr, fmt.Sprint(port)),
		&ssh.ClientConfig{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			User:            user,
			Auth:            auth,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh client:%w", err)
	}

	return &Client{sshClient: sshClient}, nil
}

// Close client net connection.
func (c *Client) Close() error {
	return c.sshClient.Close() //nolint:wrapcheck
}

func (c *Client) SetCmdLine(ctx context.Context, name string, args []string) (*Client, error) {
	mySession, err := c.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh session:%w", err)
	}
	cmd := &cmd{
		name:      name,
		args:      args,
		mySession: mySession,
		context:   ctx,
	}

	return &Client{Cmd: cmd}, nil
}

func (c *Client) RunCtx() error {
	return c.Cmd.runCtx()
}

// NewAuthKey returns auth method from private key with or without passphrase.
func NewAuthKey(keyFile string) ([]ssh.AuthMethod, error) {
	f, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read ssh key failed: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	return []ssh.AuthMethod{
		ssh.PublicKeys(signer),
	}, nil
}
