//go:build !windows

package main

import "errors"

func dpapiDecrypt([]byte) ([]byte, error) {
	return nil, errors.New("encrypted key files work on Windows only; set ANTHROPIC_API_KEY instead")
}
