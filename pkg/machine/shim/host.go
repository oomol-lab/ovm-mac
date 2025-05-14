//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package shim

import (
	"context"
	"fmt"
	"os/exec"

	"bauklotze/pkg/decompress"
	"bauklotze/pkg/machine"
	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/events"
	"bauklotze/pkg/machine/krunkit"
	"bauklotze/pkg/machine/ssh/service"
	"bauklotze/pkg/machine/vfkit"
	"bauklotze/pkg/machine/vmconfig"
	"bauklotze/pkg/machine/volumes"

	"github.com/sirupsen/logrus"
)

// Init initializes the VM provider based on the provided options.
func Init(opts *vmconfig.VMOpts) (*vmconfig.MachineConfig, error) {
	var vmp vmconfig.VMProvider
	switch opts.VMM {
	case vmconfig.KrunKit:
		vmp = krunkit.NewProvider()
	case vmconfig.VFkit:
		vmp = vfkit.NewProvider()
	default:
		return nil, fmt.Errorf("invalid VM type")
	}

	return vmp.InitializeVM(opts) //nolint:wrapcheck
}

// Update updates the VM provider with the provided options.
func Update(mc *vmconfig.MachineConfig, opts *vmconfig.VMOpts) (*vmconfig.MachineConfig, error) {
	mc.VMType = opts.VMM
	mc.Resources.CPUs = opts.CPUs
	mc.Resources.MemoryInMB = opts.MemoryInMiB
	mc.Mounts = volumes.CmdLineVolumesToMounts(opts.Volumes)

	if mc.Bootable.Version != opts.BootVersion {
		logrus.Infof("Bootable image version is not match, try to update boot image")
		if err := decompress.UncompressZSTD(opts.BootImage, mc.Bootable.Path); err != nil {
			return nil, fmt.Errorf("update boot image failed: %w", err)
		}
		mc.Bootable.Version = opts.BootVersion
	}

	if mc.DataDisk.Version != opts.DataVersion {
		logrus.Infof("Data image version is not match, try to update data image")
		if err := machine.CreateAndResizeDisk(mc.DataDisk.Path, define.DataDiskSizeInGB); err != nil {
			return nil, fmt.Errorf("update data image failed: %w", err)
		}
		mc.DataDisk.Version = opts.DataVersion
	}

	return mc, nil
}

// RaceWait waits for all the commands to finish. Return the first error.
func RaceWait(cmds ...*exec.Cmd) error {
	ctx, cancel := context.WithCancelCause(context.Background())

	for _, cmd := range cmds {
		go func(c *exec.Cmd) {
			err := c.Wait()
			cancel(fmt.Errorf("%q got exit with: %w", c.Args, err))
		}(cmd)
	}

	<-ctx.Done()
	return ctx.Err()
}

// Start starts the VM provider.
// 1. Start the network stack
// 2. Start the VM provider
// 3. Start the SSH auth and TimeSync service
func Start(parentCtx context.Context, mc *vmconfig.MachineConfig, vmp vmconfig.VMProvider) error {
	ctx, cancel := context.WithCancelCause(context.Background())
	context.AfterFunc(parentCtx, func() {
		if vmp.GetVMState().SSHReady {
			logrus.Infof("Do sync disk before shutdown")
			events.NotifyRun(events.SyncMachineDisk)
			if err := service.DoSync(mc); err != nil {
				logrus.Warnf("sync disk err: %v", err.Error())
			}
		}

		cancel(context.Cause(parentCtx))
	})
	// 1. Start the network stack
	if err := vmp.StartNetworkProvider(ctx, mc); err != nil {
		return fmt.Errorf("failed to start network stack: %w", err)
	}

	// 2. Start the VM provider
	if err := vmp.StartVMProvider(ctx, mc); err != nil {
		return fmt.Errorf("failed to start network stack: %w", err)
	}

	// Optional services are placed in separation go routines, these services will not crash the VM even if they fail
	go func() {
		logrus.Infof("Start ssh auth service")
		if err := vmp.StartSSHAuthService(ctx, mc); err != nil {
			logrus.Warnf("ssh auth service stop: %v", err)
		}
	}()

	go func() {
		if err := vmp.StartTimeSyncService(ctx, mc); err != nil {
			logrus.Warnf("time sync service stop: %v", err)
		}
	}()

	return nil
}
