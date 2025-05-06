//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"os"

	"bauklotze/pkg/machine/events"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func init() {
	loggingHook()
}

func main() {
	app := cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-out",
				Usage: "where to write the log, support file, stdout, default is file( in workspace log dir )",
				Value: "file",
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

func loggingHook() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		ForceColors:     true,
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	logrus.SetOutput(os.Stderr)
}
