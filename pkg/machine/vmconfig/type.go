package vmconfig

import (
	"runtime"
)

const (
	KrunKit = "krunkit"
	VFkit   = "vfkit"
)

func GetVMM() string {
	if runtime.GOARCH == "amd64" {
		return VFkit
	}
	return KrunKit
}
