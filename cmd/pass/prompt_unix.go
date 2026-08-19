//go:build darwin || dragonfly || freebsd || netbsd || openbsd || solaris

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func readPassword(label string) (string, error) {
	terminal, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("cannot securely prompt without a controlling terminal: %w", err)
	}
	defer terminal.Close()

	stty, err := exec.LookPath("stty")
	if err != nil {
		return "", fmt.Errorf("cannot securely prompt because stty is unavailable: %w", err)
	}
	original, err := runStty(stty, terminal, "-g")
	if err != nil {
		return "", fmt.Errorf("cannot read terminal settings: %w", err)
	}
	if _, err := runStty(stty, terminal, "-echo"); err != nil {
		return "", fmt.Errorf("cannot disable terminal echo: %w", err)
	}
	defer func() { _, _ = runStty(stty, terminal, strings.TrimSpace(original)) }()

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

func runStty(path string, terminal *os.File, argument string) (string, error) {
	command := exec.Command(path, argument)
	command.Stdin = terminal
	command.Stderr = terminal
	output, err := command.Output()
	return string(output), err
}
