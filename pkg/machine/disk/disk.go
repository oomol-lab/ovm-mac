//  SPDX-FileCopyrightText: 2024-2026 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package disk

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

//go:embed source.ext4.tar
var sourceCodeExt4Disk []byte

func ExtractSourceCodeDisk(ctx context.Context, targetDirPath string, overwrite bool) error {
	_, err := os.Stat(filepath.Join(targetDirPath, "source.ext4"))
	if err == nil && !overwrite {
		logrus.Infof("source code disk already exists, skip extraction")
		return nil
	}

	if err := os.MkdirAll(targetDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	cmd := exec.CommandContext(ctx, "tar", "-xaS", "-C", targetDirPath, "-f", "-", "source.ext4")
	cmd.Stdin = bytes.NewReader(sourceCodeExt4Disk)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	logrus.Infof("cmdline: %q", cmd.Args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar extraction failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}
