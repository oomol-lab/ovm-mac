//  SPDX-FileCopyrightText: 2024-2025 OOMOL, Inc. <https://www.oomol.com>
//  SPDX-License-Identifier: MPL-2.0

package ssh

import (
	"fmt"
	"time"

	"bauklotze/pkg/machine/define"
	"bauklotze/pkg/machine/vmconfig"
)

func GetKernelInfo(mc *vmconfig.MachineConfig) error {
	return Run(mc.SSH.IdentityPath, define.LocalHostURL, define.DefaultUserInVM, uint(mc.SSH.Port), "uname", []string{
		"-a",
	})
}

func DoTimeSync(mc *vmconfig.MachineConfig) error {
	return Run(mc.SSH.IdentityPath, define.LocalHostURL, define.DefaultUserInVM, uint(mc.SSH.Port), "date", []string{
		"-s",
		fmt.Sprintf("@%d", time.Now().Unix()),
	})
}

func DoSync(mc *vmconfig.MachineConfig) error {
	return Run(mc.SSH.IdentityPath, define.LocalHostURL, define.DefaultUserInVM, uint(mc.SSH.Port), "bash", []string{
		"-c",
		"sync",
	})
}
