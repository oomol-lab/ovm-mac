//  SPDX-FileCopyrightText: 2024-2024 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package pty

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

const (
	rows = uint16(80)
	cols = uint16(80)
)

func RunInPty(c *exec.Cmd) (*os.File, error) {
	return pty.StartWithAttrs(c, &pty.Winsize{ //nolint:wrapcheck
		Cols: cols,
		Rows: rows,
	}, &syscall.SysProcAttr{})
}
