//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func readPassword(label string) (string, error) {
	terminal, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("cannot securely prompt without a controlling terminal: %w", err)
	}
	defer terminal.Close()

	state, err := terminalState(terminal.Fd(), syscall.TCGETS, nil)
	if err != nil {
		return "", fmt.Errorf("cannot read terminal settings: %w", err)
	}
	noEcho := *state
	noEcho.Lflag &^= syscall.ECHO
	if _, err := terminalState(terminal.Fd(), syscall.TCSETS, &noEcho); err != nil {
		return "", fmt.Errorf("cannot disable terminal echo: %w", err)
	}
	defer func() { _, _ = terminalState(terminal.Fd(), syscall.TCSETS, state) }()

	if _, err := fmt.Fprint(terminal, label); err != nil {
		return "", err
	}
	line, readErr := bufio.NewReader(terminal).ReadString('\n')
	fmt.Fprintln(terminal)
	if readErr != nil && len(line) == 0 {
		return "", fmt.Errorf("cannot read master password: %w", readErr)
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func terminalState(fd uintptr, request uintptr, replacement *syscall.Termios) (*syscall.Termios, error) {
	state := replacement
	if state == nil {
		state = new(syscall.Termios)
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, uintptr(unsafe.Pointer(state)))
	if errno != 0 {
		return nil, errno
	}
	return state, nil
}
