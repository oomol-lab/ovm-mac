//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"os"

	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/events"
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
			return ctx, nil
		},
	}

	NotifyAndExit(app.Run(context.Background(), os.Args))
}

func NotifyAndExit(err error) {
	exitCode := 0
	if err != nil {
		exitCode = 1
		logrus.Error(err.Error())
		events.NotifyError(err)
	}
	events.NotifyExit()
	logrus.Exit(exitCode)
}
