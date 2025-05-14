//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"bauklotze/pkg/api/server"
	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/events"
	"bauklotze/pkg/machine/krunkit"
	"bauklotze/pkg/machine/shim"
	"bauklotze/pkg/machine/vfkit"
	"bauklotze/pkg/machine/vmconfig"
	"bauklotze/pkg/registry"
	"bauklotze/pkg/system"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

var startCmd = cli.Command{
	Name:   "start",
	Usage:  "Start a virtual machine",
	Action: start,
	Before: func(ctx context.Context, cli *cli.Command) (context.Context, error) {
		events.CurrentStage = events.Run
		return ctx, nil
	},
}

func start(parentCtx context.Context, cli *cli.Command) error {
	opts := &vmconfig.VMOpts{
		VMName: cli.String("name"),
		PPID:   cli.Int("ppid"),
	}

	vmcFile := filepath.Join(vmconfig.Workspace, define.ConfigPrefixDir, fmt.Sprintf("%s.json", opts.VMName))

	// We first check the status of the pid passed in via --ppid,
	// and if it is inactive, exit immediately without running any of the following code
	isRunning, err := system.IsProcessAlive(int(opts.PPID))
	if !isRunning {
		return fmt.Errorf("PPID %d exited, possible error: %w", opts.PPID, err)
	}

	events.NotifyRun(events.LoadMachineConfig)
	mc, err := vmconfig.LoadMachineFromPath(vmcFile)
	if err != nil {
		return fmt.Errorf("load machine config file failed: %w", err)
	}

	g, ctx := errgroup.WithContext(parentCtx)

	// WatchPPID
	g.Go(func() error {
		const tickerInterval = 300 * time.Millisecond
		ticker := time.NewTicker(tickerInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				isRunning, err := system.IsProcessAlive(int(opts.PPID))
				if !isRunning {
					return fmt.Errorf("PPID %d exited, possible error: %w", opts.PPID, err)
				}
			}
		}
	})
	// Listen signal
	g.Go(func() error {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case s := <-sigChan:
			return fmt.Errorf("catch signal: %v", s.String())
		}
	})

	// Start Rest API
	g.Go(func() error {
		endPoint := mc.RestAPISocks
		logrus.Infof("Start rest api service at %q", endPoint)
		return server.RestService(ctx, mc, endPoint)
	})

	// start machine
	g.Go(func() error {
		var vmp vmconfig.VMProvider
		switch mc.VMType {
		case vmconfig.KrunKit:
			vmp = krunkit.NewProvider()
		case vmconfig.VFkit:
			vmp = vfkit.NewProvider()
		default:
			return fmt.Errorf("invalid vmm type")
		}

		if err := shim.Start(ctx, mc, vmp); err != nil {
			return fmt.Errorf("start machine %q error: %w", mc.VMName, err)
		}

		// NOTE:
		// shim.Wait do not need to support context.
		// once the parent context is done, the cmd will be killed by the os.Process.Kill(). so the shim.Wait will return immediately.
		return shim.RaceWait(registry.GetCmds()...)
	})

	return g.Wait() //nolint:wrapcheck
}
