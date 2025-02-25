//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

//go:build darwin

package krunkit

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"bauklotze/pkg/machine/events"
	"bauklotze/pkg/machine/helper"
	"bauklotze/pkg/machine/vmconfig"

	"bauklotze/pkg/machine/ignition"
	mypty "bauklotze/pkg/pty"

	vfConfig "github.com/crc-org/vfkit/pkg/config"
	"github.com/sirupsen/logrus"
)

const applehvMACAddress = "5a:94:ef:e4:0c:ee"

// setupKrunkitDevices add devices into VirtualMachine
func setupDevices(mc *vmconfig.MachineConfig) ([]vfConfig.VirtioDevice, error) {
	var devices []vfConfig.VirtioDevice

	disk, err := vfConfig.VirtioBlkNew(mc.Bootable.Image.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to create bootable disk device: %w", err)
	}
	rng, err := vfConfig.VirtioRngNew()
	if err != nil {
		return nil, fmt.Errorf("failed to create rng device: %w", err)
	}

	// externalDisk is the disk used to store the user data, it will format as ext4
	externalDisk, err := vfConfig.VirtioBlkNew(mc.DataDisk.Image.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to create externalDisk device: %w", err)
	}

	// using gvproxy as network backend
	netDevice, err := vfConfig.VirtioNetNew(applehvMACAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create net device: %w", err)
	}
	gvproxySocket, err := mc.GVProxyNetworkBackendSocks()
	if err != nil {
		return nil, fmt.Errorf("failed to get gvproxy socket: %w", err)
	}
	netDevice.SetUnixSocketPath(gvproxySocket.GetPath())

	// externalDisk **must** be at the end of the device
	devices = append(devices, disk, rng, netDevice, externalDisk)

	VirtIOMounts, err := helper.VirtIOFsToVFKitVirtIODevice(mc.Mounts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert virtio fs to virtio device: %w", err)
	}
	devices = append(devices, VirtIOMounts...)

	return devices, nil
}

// Taken from vfConfig.VirtualMachine
func createCmd(ctx context.Context, vfkitPath string, vm *vfConfig.VirtualMachine) (*exec.Cmd, error) {
	args, err := vm.ToCmdLine()
	if err != nil {
		return nil, fmt.Errorf("createCmd err:%w", err)
	}

	cmd := exec.CommandContext(ctx, vfkitPath, args...)
	cmd.ExtraFiles = extraFiles(vm)

	return cmd, nil
}

// Taken from vfConfig.VirtualMachine
func extraFiles(vm *vfConfig.VirtualMachine) []*os.File {
	extraFiles := []*os.File{}
	for _, dev := range vm.Devices {
		virtioNet, ok := dev.(*vfConfig.VirtioNet)
		if !ok {
			continue
		}
		if virtioNet.Socket != nil {
			extraFiles = append(extraFiles, virtioNet.Socket)
		}
	}
	return extraFiles
}

func startKrunkit(ctx context.Context, mc *vmconfig.MachineConfig) error {
	bootloader := mc.AppleKrunkitHypervisor.Krunkit.VirtualMachine.Bootloader
	if bootloader == nil {
		return fmt.Errorf("unable to determine boot loader for this machine")
	}

	vmc := vfConfig.NewVirtualMachine(uint(mc.Resources.CPUs), uint64(mc.Resources.Memory), bootloader)

	defaultDevices, err := setupDevices(mc)
	if err != nil {
		return fmt.Errorf("failed to get default devices: %w", err)
	}
	vmc.Devices = append(vmc.Devices, defaultDevices...)

	krunkitBin := mc.Dirs.Hypervisor.Bin.GetPath()
	logrus.Infof("krunkit binary path is: %s", krunkitBin)

	krunkitCmd, err := createCmd(ctx, krunkitBin, vmc)
	if err != nil {
		return fmt.Errorf("failed to create krunkit command: %w", err)
	}
	libsDir := mc.Dirs.Hypervisor.LibsDir.GetPath()
	krunkitCmd.Env = append(krunkitCmd.Env, fmt.Sprintf("DYLD_LIBRARY_PATH=%s", libsDir))

	// Add the "krun-log-level" allflag for setting up the desired log level for libkrun's debug facilities.
	// Log level for libkrun (0=off, 1=error, 2=warn, 3=info, 4=debug, 5 or higher=trace)
	krunkitCmd.Args = append(krunkitCmd.Args, "--krun-log-level", "3")

	err = ignition.GenerateIgnScripts(mc)
	if err != nil {
		return fmt.Errorf("failed to generate ignition scripts: %w", err)
	}

	logrus.Infof("FULL KRUNKIT CMDLINE:%q", krunkitCmd.Args)
	events.NotifyRun(events.StartVMProvider, "krunkit staring...")

	// Run krunkit in pty, the pty should never close because the krunkit is a background process
	ptmx, err := mypty.RunInPty(krunkitCmd)
	if err != nil {
		return fmt.Errorf("failed to run krunkit in pty: %w", err)
	}
	mc.VmmCmd = krunkitCmd

	go func() {
		_, _ = io.Copy(os.Stdout, ptmx)
	}()

	events.NotifyRun(events.StartVMProvider, "krunkit started")

	return nil
}
