//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package machine

import (
	"context"
	"fmt"
	"os"
	"time"

	"bauklotze/pkg/decompress"
	"bauklotze/pkg/httpclient"
	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/events"
	"bauklotze/pkg/machine/fs"
	"bauklotze/pkg/machine/ignition"
	sshService "bauklotze/pkg/machine/ssh/service"
	"bauklotze/pkg/machine/vmconfig"
	"bauklotze/pkg/machine/volumes"

	"github.com/containers/common/pkg/strongunits"
	vfConfig "github.com/crc-org/vfkit/pkg/config"
	"github.com/prashantgupta24/mac-sleep-notifier/notifier"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const defaultPingTimeout = 5 * time.Second
const defaultPingInterval = 200 * time.Millisecond

func WaitPodmanReady(ctx context.Context, sock string) error {
	client := httpclient.New().SetTransport(httpclient.CreateUnixTransport(sock))
	timeout := time.After(defaultPingTimeout)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cancel WaitPodmanReady, ctx has been done: %w", context.Cause(ctx))
		case <-timeout:
			return fmt.Errorf("timeout reached while waiting for Podman API")
		default:
			logrus.Info("Try ping Podman API")
			time.Sleep(defaultPingInterval)

			if err := client.Get("_ping"); err == nil {
				logrus.Infof("Podman ping test success")
				return nil
			}
		}
	}
}

var (
	defaultBackoff = 100 * time.Millisecond
	maxTried       = 100
)

func WaitSSHStarted(ctx context.Context, mc *vmconfig.MachineConfig) bool {
	for range maxTried {
		if ctx.Err() != nil {
			logrus.Warnf("cancel WaitSSHStarted, ctx has been done: %v", context.Cause(ctx))
			return false
		}

		if err := sshService.GetKernelInfo(ctx, mc); err != nil {
			logrus.Warnf("SSH readiness check err: %v, try again", err)
			time.Sleep(defaultBackoff)
			continue
		}
		return true
	}
	return false
}

// InitializeVM initialize the data and boot image and write the machine config.
// both the vfkit and krunkit using the same init logic
func InitializeVM(opts *vmconfig.VMOpts) (*vmconfig.MachineConfig, error) {
	mc := vmconfig.NewMachineConfig(opts)

	if err := mc.GetSSHPort(); err != nil {
		return nil, fmt.Errorf("failed to get ssh port: %w", err)
	}

	if err := mc.MakeDirs(); err != nil {
		return nil, fmt.Errorf("make work space err: %w", err)
	}

	if err := mc.CreateSSHKey(); err != nil {
		return nil, fmt.Errorf("create ssh key err: %w", err)
	}

	logrus.Infof("Decompress %q to %q", opts.BootImage, mc.Bootable.Path)

	events.NotifyInit(events.ExtractBootImage)
	if err := decompress.UncompressZSTD(opts.BootImage, mc.Bootable.Path); err != nil {
		return nil, fmt.Errorf("initialize vm failed: %w", err)
	}

	logrus.Infof("create data disk image %q with sizeInGb %d", mc.DataDisk.Path, define.DataDiskSizeInGB)
	if err := CreateAndResizeDisk(mc.DataDisk.Path, define.DataDiskSizeInGB, false); err != nil {
		return nil, fmt.Errorf("initialize vm failed: %w", err)
	}

	return mc, nil
}

const applehvMACAddress = "5a:94:ef:e4:0c:ee"

// SetupDevices add devices into VirtualMachine
func SetupDevices(mc *vmconfig.MachineConfig) ([]vfConfig.VirtioDevice, error) {
	var devices []vfConfig.VirtioDevice

	disk, err := vfConfig.VirtioBlkNew(mc.Bootable.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create bootable disk device: %w", err)
	}
	rng, err := vfConfig.VirtioRngNew()
	if err != nil {
		return nil, fmt.Errorf("failed to create rng device: %w", err)
	}

	// externalDisk is the disk used to store the user data, it will format as ext4
	externalDisk, err := vfConfig.VirtioBlkNew(mc.DataDisk.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create externalDisk device: %w", err)
	}

	// using gvproxy as network backend
	netDevice, err := vfConfig.VirtioNetNew(applehvMACAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create net device: %w", err)
	}

	netDevice.SetUnixSocketPath(mc.GetNetworkStackEndpoint())

	// externalDisk **must** be at the end of the device
	devices = append(devices, disk, rng, netDevice, externalDisk)

	VirtIOMounts, err := VirtIOFsToVFKitVirtIODevice(mc.Mounts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert virtio fs to virtio device: %w", err)
	}
	devices = append(devices, VirtIOMounts...)

	return devices, nil
}

// CreateDynamicConfigure create a dynamic machine configure (bootloader, mounts, devices) from vmconfig.MachineConfig, which is used to create a virtual machine.
func CreateDynamicConfigure(mc *vmconfig.MachineConfig) (*vfConfig.VirtualMachine, error) {
	bootloaderConfig := vfConfig.NewEFIBootloader(fmt.Sprintf("%s/efi-bootloader.img", mc.Dirs.DataDir), true)
	dynamicVMConfig := vfConfig.NewVirtualMachine(uint(mc.Resources.CPUs), uint64(mc.Resources.MemoryInMB), bootloaderConfig)
	defaultDevices, err := SetupDevices(mc)
	if err != nil {
		return nil, fmt.Errorf("failed to get default devices: %w", err)
	}

	dynamicVMConfig.Devices = append(dynamicVMConfig.Devices, defaultDevices...)

	if err = ignition.GenerateScripts(mc); err != nil {
		return nil, fmt.Errorf("failed to generate ignition scripts: %w", err)
	}

	return dynamicVMConfig, nil
}

// SyncTimeOnWake start Sleep Notifier and dispatch tasks
func SyncTimeOnWake(ctx context.Context, mc *vmconfig.MachineConfig) error { //nolint: staticcheck
	notifierCh := notifier.GetInstance().Start()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cancel SleepNotifier, ctx has been done: %w", context.Cause(ctx))
		case activity := <-notifierCh:
			if activity.Type == notifier.Awake {
				logrus.Infof("host awake, do time sync for vm")
				if err := sshService.DoTimeSync(ctx, mc); err != nil {
					logrus.Warnf("Failed to sync timestamp: %v", err)
				}
			}
		}
	}
}

func VirtIOFsToVFKitVirtIODevice(mounts []volumes.Mount) ([]vfConfig.VirtioDevice, error) {
	virtioDevices := make([]vfConfig.VirtioDevice, 0, len(mounts))
	for _, vol := range mounts {
		virtfsDevice, err := vfConfig.VirtioFsNew(vol.Source, vol.Tag)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtio fs device: %w", err)
		}
		virtioDevices = append(virtioDevices, virtfsDevice)
	}
	return virtioDevices, nil
}

// CreateAndResizeDisk create a disk file with sizeInGB, and truncate it to sizeInGB.
func CreateAndResizeDisk(f string, sizeInGB int64, force bool) error {
	disk := fs.NewFile(f)
	if force {
		if err := disk.Delete(); err != nil {
			return fmt.Errorf("failed to delete %q: %w", f, err)
		}
	}

	if disk.IsExist() {
		logrus.Infof("data disk %q already exists, skip re-create data disk", f)
		return nil
	}

	logrus.Infof("creating data disk %q and resize to %d GB", f, sizeInGB)
	file, err := os.OpenFile(f, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create disk: %q, %w", f, err)
	}
	defer file.Close() //nolint:errcheck

	if err = os.Truncate(f, int64(strongunits.GiB(sizeInGB).ToBytes())); err != nil {
		return fmt.Errorf("failed to truncate disk: %w", err)
	}

	return nil
}

func StartSSHAuthService(ctx context.Context, mc *vmconfig.MachineConfig) error {
	sshAuthService := sshService.NewSSHAuthService(
		mc.SSHAuthSocks.LocalSocks,
		mc.SSHAuthSocks.RemoteSocks,
		mc.SSH.RemoteUsername,
		mc.SSH.PrivateKeyPath,
		mc.SSH.Port,
	)

	g, ctx2 := errgroup.WithContext(ctx)
	g.Go(func() error {
		return sshAuthService.StartSSHAuthServiceAndForwardV2(ctx2)
	})

	g.Go(func() error {
		return sshAuthService.StartUnixSocketForward(ctx2)
	})

	return g.Wait() //nolint:wrapcheck
}
