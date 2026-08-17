package main

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type clipboardCommand struct {
	program string
	args    []string
}

func copyToClipboard(value string) error {
	commands := clipboardCommands()
	var failures []error
	for _, candidate := range commands {
		path, err := exec.LookPath(candidate.program)
		if err != nil {
			continue
		}
		command := exec.Command(path, candidate.args...)
		command.Stdin = strings.NewReader(value)
		if output, err := command.CombinedOutput(); err == nil {
			return nil
		} else if len(output) > 0 {
			failures = append(failures, fmt.Errorf("%s: %w: %s", candidate.program, err, output))
		} else {
			failures = append(failures, fmt.Errorf("%s: %w", candidate.program, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("clipboard command failed: %w", errors.Join(failures...))
	}
	return errors.New("clipboard unavailable: install wl-copy, xclip, xsel, pbcopy, or clip.exe")
}

func clipboardCommands() []clipboardCommand {
	switch runtime.GOOS {
	case "darwin":
		return []clipboardCommand{{program: "pbcopy"}}
	case "windows":
		return []clipboardCommand{{program: "clip.exe"}}
	default:
		return []clipboardCommand{
			{program: "wl-copy"},
			{program: "xclip", args: []string{"-selection", "clipboard", "-in"}},
			{program: "xsel", args: []string{"--clipboard", "--input"}},
		}
	}
}
