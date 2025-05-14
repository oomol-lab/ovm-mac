//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"os"
	"path/filepath"

	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/events"
	"bauklotze/pkg/machine/io"
	"bauklotze/pkg/machine/vmconfig"

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
				Usage: "where to write the log, support file, stdout, default is file",
				Value: define.LogOutFile,
			},
			&cli.StringFlag{
				Name:  "report-url",
				Usage: "URL to send report events to",
			},
			&cli.IntFlag{
				Name:  "ppid",
				Usage: "Parent process id, if not given, the ppid is the current process's ppid",
				Value: int64(os.Getppid()),
			},
		},
		Commands: []*cli.Command{
			&initCmd,
			&startCmd,
		},
		Before: func(ctx context.Context, command *cli.Command) (context.Context, error) {
			events.SetReportURL(command.String("report-url"))
			vmconfig.Workspace = command.String("workspace")
			loggerSetup(command.String("log-out"), command.String("workspace"))
			return ctx, nil
		},
	}

	notifyAndExit(app.Run(context.Background(), os.Args))
}

func notifyAndExit(err error) {
	exitCode := 0
	if err != nil {
		exitCode = 1
		logrus.Error(err.Error())
		events.NotifyError(err)
	}
	events.NotifyExit()
	logrus.Exit(exitCode)
}

const MaxSizeInMB = 5

// loggerSetup outType: file, stdout
// if outType is file, workspace is required
// if outType is terminal, workspace is not required, all output will be sent to Terminal's stdout/stderr
func loggerSetup(outType string, workspace string) {
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
