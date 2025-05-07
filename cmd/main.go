//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"os"

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
