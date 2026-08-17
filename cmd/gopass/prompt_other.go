//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !solaris && !windows

package main

import "errors"

func readPassword(_ string) (string, error) {
	return "", errors.New("secure terminal prompting is not supported on this operating system; use LESSPASS_MASTER_PASSWORD")
}
