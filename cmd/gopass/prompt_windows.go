//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const enableEchoInput = 0x0004

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode = kernel32.NewProc("GetConsoleMode")
	setConsoleMode = kernel32.NewProc("SetConsoleMode")
)

func readPassword(label string) (string, error) {
	input, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("cannot securely prompt without a console: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	if err != nil {
		return "", fmt.Errorf("cannot open the console for prompting: %w", err)
	}
	defer output.Close()

	var original uint32
	if err := windowsCall(getConsoleMode, input.Fd(), uintptr(unsafe.Pointer(&original))); err != nil {
		return "", fmt.Errorf("cannot read console settings: %w", err)
	}
	if err := windowsCall(setConsoleMode, input.Fd(), uintptr(original&^enableEchoInput)); err != nil {
		return "", fmt.Errorf("cannot disable console echo: %w", err)
	}
	defer func() { _ = windowsCall(setConsoleMode, input.Fd(), uintptr(original)) }()

	if _, err := fmt.Fprint(output, label); err != nil {
		return "", err
	}
	line, readErr := bufio.NewReader(input).ReadString('\n')
	fmt.Fprintln(output)
	if readErr != nil && len(line) == 0 {
		return "", fmt.Errorf("cannot read master password: %w", readErr)
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func windowsCall(procedure *syscall.LazyProc, arguments ...uintptr) error {
	result, _, callErr := procedure.Call(arguments...)
	if result != 0 {
		return nil
	}
	if callErr != nil && callErr != syscall.Errno(0) {
		return callErr
	}
	return syscall.EINVAL
}
