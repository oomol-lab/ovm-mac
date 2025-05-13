package vmconfig

import (
	"runtime"

	"github.com/sirupsen/logrus"
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
	logrus.Info("vm provider is: ", provider)
	return provider
}
