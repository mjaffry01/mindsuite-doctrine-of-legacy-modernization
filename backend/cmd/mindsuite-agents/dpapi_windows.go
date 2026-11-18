//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"
)

// Windows DPAPI: the key file made by Export-Clixml can only be decrypted by the same Windows user on the same machine.
type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	procUnprotectData = syscall.NewLazyDLL("crypt32.dll").NewProc("CryptUnprotectData")
	procLocalFree     = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")
)

const cryptprotectUIForbidden = 0x1

func dpapiDecrypt(blob []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, errors.New("the encrypted value is empty")
	}
	in := dataBlob{cbData: uint32(len(blob)), pbData: &blob[0]}
	var out dataBlob
	r, _, err := procUnprotectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, cryptprotectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	plain := make([]byte, out.cbData)
	copy(plain, unsafe.Slice(out.pbData, out.cbData))
	return plain, nil
}
