//  SPDX-FileCopyrightText: 2024-2024 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0
//go:build darwin

package ignition

import (
	"fmt"
	"path/filepath"

	"bauklotze/pkg/machine/fs"
	"bauklotze/pkg/machine/vmconfig"
)

func GenerateScripts(mc *vmconfig.MachineConfig) error {
	var ignScriptFile = filepath.Join("/", "tmp", "initfs", "ovm_ign.sh")
	ign := NewIgnitionBuilder(
		&DynamicIgnitionV3{
			CodeBuffer:      nil,
			File:            fs.NewFile(ignScriptFile),
			VMType:          vmconfig.KrunKit,
			Mounts:          mc.Mounts,
			SSHIdentityPath: fs.NewFile(mc.SSH.PrivateKeyPath),
		})

	err := ign.GenerateConfig()
	if err != nil {
		return fmt.Errorf("failed to generate ignition config: %w", err)
	}

	err = ign.Write()
	if err != nil {
		return fmt.Errorf("failed to write ignition file: %w", err)
	}

	return nil
}
