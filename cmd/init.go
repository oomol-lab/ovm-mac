//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"

	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/events"
	"bauklotze/pkg/machine/fs"
	"bauklotze/pkg/machine/shim"
	"bauklotze/pkg/machine/vmconfig"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var initCmd = cli.Command{
	DisableSliceFlagSeparator: true,
	Name:                      "init",
	Usage:                     "Initialize a new virtual machine",
	Action:                    initMachine,
	Before: func(ctx context.Context, command *cli.Command) (context.Context, error) {
		events.CurrentStage = events.Init
		return ctx, nil
	},

	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "cpus",
			Usage: "Number of CPUs to allocate to the VM",
			Value: int64(math.Min(float64(runtime.NumCPU()-1), 8)), //nolint:mnd
		},
		&cli.IntFlag{
			Name:  "memory",
			Usage: "Amount of memory (in MB) to allocate to the VM",
			Value: 4096, //nolint:mnd
		},
		&cli.StringSliceFlag{
			Name:    "volume",
			Usage:   "Host directory to mount into the VM",
			Aliases: []string{"v"},
		},
		&cli.StringFlag{
			Name:     "boot",
			Usage:    "Boot image to use for the VM",
			Aliases:  []string{"b"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "boot-version",
			Usage:    "version field the control boot image should be re-initialized, if the given version is not equal to the current version, re-initialize the boot image",
			Value:    "v1.0",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "data-version",
			Usage:    "version field the control data image should be re-initialized, if the given version is not equal to the current version, re-initialize the data image",
			Value:    "v1.0",
			Required: true,
		},

		&cli.StringFlag{
			Name:  "vmm",
			Usage: "vm provider, support: krunkit, vfkit",
			Value: vmconfig.GetVMM(),
		},
	},
}

func initMachine(ctx context.Context, cli *cli.Command) error {
	opts := &vmconfig.VMOpts{
		VMName:      cli.String("name"),
		Workspace:   cli.String("workspace"),
		PPID:        cli.Int("ppid"),
		CPUs:        cli.Int("cpus"),
		MemoryInMiB: cli.Int("memory"),
		Volumes:     cli.StringSlice("volume"),
		BootImage:   cli.String("boot"),
		BootVersion: cli.String("boot-version"),
		DataVersion: cli.String("data-version"),
		VMM:         cli.String("vmm"),
	}

	migrateData(opts)

	// add a default mount point that store generated ignition scripts
	opts.Volumes = append(opts.Volumes, define.IgnMnt)

	vmcFile := opts.GetVMConfigPath()

	var reinit bool
	mc, err := vmconfig.LoadMachineFromPath(vmcFile)
	if err != nil {
		// set reinit flag to true means the vm need to be full reset
		reinit = true
	}

	if reinit {
		events.NotifyInit(events.InitNewMachine)
		mc, err = shim.Init(opts)
	} else {
		events.NotifyInit(events.InitUpdateConfig)
		mc, err = shim.Update(mc, opts)
	}

	if err != nil {
		return fmt.Errorf("init machine failed: %w", err)
	}

	if err := mc.Write(); err != nil {
		return fmt.Errorf("write machine config file failed: %w", err)
	}

	events.NotifyInit(events.InitSuccess)

	return nil
}

func migrateData(opts *vmconfig.VMOpts) {
	if err := os.RemoveAll(filepath.Join(opts.Workspace, "logs")); err != nil {
		logrus.Errorf("remove logs failed: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(opts.Workspace, "pids")); err != nil {
		logrus.Errorf("remove pids failed: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(opts.Workspace, "config")); err != nil {
		logrus.Errorf("remove pids failed: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(opts.Workspace, "socks")); err != nil {
		logrus.Errorf("remove pids failed: %v", err)
	}

	f := filepath.Join(opts.Workspace, "data", "libkrun", "default-arm64-data.raw")
	if runtime.GOARCH == "amd64" {
		f = filepath.Join(opts.Workspace, "data", "vfkit", "default-amd64-data.raw")
	}
	oldDataDiskFile := fs.NewFile(f)

	f = filepath.Join(opts.Workspace, opts.VMName, define.DataPrefixDir, "data.img")
	newDataDiskFile := fs.NewFile(f)

	if !oldDataDiskFile.IsExist() {
		logrus.Info("old data disk does not exist, no need to migrate")
		return
	}

	logrus.Infof("Move old data disk %q file to %q", oldDataDiskFile.GetPath(), newDataDiskFile.GetPath())
	if err := os.MkdirAll(filepath.Dir(newDataDiskFile.GetPath()), 0755); err != nil {
		logrus.Errorf("mkdir failed: %v", err)
	}

	if err := os.Rename(oldDataDiskFile.GetPath(), newDataDiskFile.GetPath()); err != nil {
		logrus.Warnf("move old data disk file failed: %v", err)
	}

	if err := os.RemoveAll(filepath.Join(opts.Workspace, "data")); err != nil {
		logrus.Errorf("remove data failed: %v", err)
	}
}
