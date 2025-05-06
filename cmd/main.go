//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"os"
	"path/filepath"

	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/events"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "workspace",
				Usage:    "workspace directory",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "name",
				Usage:    "machine name",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "log-out",
				Usage: "where to write the log, support file, stdout, default is file( in workspace log dir )",
				Value: define.LogOutFile,
			},
			&cli.IntFlag{
				Name:  "ppid",
				Usage: "Parent process id, if not given, the ppid is the current process's ppid",
				Value: int64(os.Getpid()),
			},
		},
		Commands: []*cli.Command{
			&initCmd,
			&startCmd,
		},
	}

	NotifyAndExit(app.Run(context.Background(), os.Args))
}

func NotifyAndExit(err error) {
	retCode := 0
	if err != nil {
		retCode = 1
		logrus.Error(err.Error())
		events.NotifyError(err)
	}
	events.NotifyExit()
	logrus.Exit(retCode)
}

// outType: file, stdout
// if outType is file, workspace is required
// if outType is terminal, workspace is not required, all output will be sent to Terminal's stdout/stderr
func loggingHook(outType string, workspace string) {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		ForceColors:     true,
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})

	logrus.SetOutput(os.Stderr)

	if outType == define.LogOutFile {
		logrus.Infof("Save log to %q", filepath.Join(workspace, define.LogPrefixDir, define.LogFileName))
		if fd, err := os.OpenFile(filepath.Join(workspace, define.LogPrefixDir, define.LogFileName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm); err == nil {
			os.Stdout = fd
			os.Stderr = fd
			logrus.SetOutput(fd)
		}
	}
}
