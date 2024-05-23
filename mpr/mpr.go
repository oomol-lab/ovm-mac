package mpr

import (
	"syscall"
)

type _UNIVERSAL_NAME_INFOA struct {
	LPSTR *[syscall.MAX_LONG_PATH]byte
}
type _REMOTE_NAME_INFOW struct {
	lpUniversalName *uint16
}

const (
	UNIVERSAL_NAME_INFO_LEVEL = 0x00000001
	REMOTE_NAME_INFO_LEVEL    = 0x00000002
)

func CStr(str string) *uint16 {
	s, _ := syscall.UTF16PtrFromString(str)
	return s
}

func WNetGetUniversalNameW(lpLocalPath string) {

	lpBuffer := new(_REMOTE_NAME_INFOW)
	size := uint32(1024)
	wNetGetUniversalNameW(CStr(lpLocalPath), uintptr(UNIVERSAL_NAME_INFO_LEVEL), lpBuffer, &size)

}

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zmpr.go mpr.go
//sys wNetGetUniversalNameW(lpLocalPath *uint16,dwInfoLevel uintptr,lpBuffer *_REMOTE_NAME_INFOW,lpBufferSize *uint32) (e error) = mpr.WNetGetUniversalNameW
