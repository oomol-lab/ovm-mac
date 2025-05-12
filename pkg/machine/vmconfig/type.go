package vmconfig

import "runtime"

const (
	KrunKit = "krunkit"
	VFkit   = "vfkit"
)

func GetVMM() string {
	provider := KrunKit
	if runtime.GOARCH == "x86_64" {
		provider = VFkit
	}
	return provider
}
