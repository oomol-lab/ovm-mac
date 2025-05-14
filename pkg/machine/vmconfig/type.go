package vmconfig

import (
	"runtime"
)

const (
	KrunKit = "krunkit"
	VFkit   = "vfkit"
)

func GetVMM() string {
	provider := KrunKit
	if runtime.GOARCH == "amd64" {
		provider = VFkit
	}
	return provider
}
