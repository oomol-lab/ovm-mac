//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package log

import (
	"os"
	"path/filepath"

	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/io"

	"github.com/sirupsen/logrus"
)

const MaxSizeInMB = 5

// Setup outType: file, stdout
// if outType is file, workspace is required
// if outType is terminal, workspace is not required, all output will be sent to Terminal's stdout/stderr
func Setup(outType string, workspace string) {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		ForceColors:     true,
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})

	logrus.SetOutput(os.Stderr)

	if outType == define.LogOutFile {
		logFile := io.NewFile(filepath.Join(workspace, define.LogPrefixDir, define.LogFileName))
		if logFile.IsExist() {
			logrus.Infof("Log file %q already exists, discarding the first 5 Mib", filepath.Join(workspace, define.LogPrefixDir, define.LogFileName))
			if err := logFile.DiscardBytesAtBegin(MaxSizeInMB); err != nil {
				logrus.Warnf("failed to discard log file: %q", err)
			}
		}

		logrus.Infof("Save log to %q", filepath.Join(workspace, define.LogPrefixDir, define.LogFileName))
		if fd, err := os.OpenFile(filepath.Join(workspace, define.LogPrefixDir, define.LogFileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm); err == nil {
			os.Stdout = fd
			os.Stderr = fd
			logrus.SetOutput(fd)
		}
	}
}
