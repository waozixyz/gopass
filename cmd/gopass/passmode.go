package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

const passUsage = `pass - generate a LessPass password and copy it to the clipboard

Usage:
  pass [flags] <profile> <site>

The profile name selects a login and settings from the gopass profiles file.
You will be prompted (hidden input) for the passphrase of your OpenSSL
secret vault when one is configured. The password is copied to the
CLIPBOARD selection; it is never displayed.

Flags:
`

// translatePassArguments rewrites the legacy two-argument `pass PROFILE SITE`
// command line into the equivalent gopass arguments, so a copy or symlink of
// the gopass binary named "pass" keeps working as the personal shorthand.
// It reports false (after printing usage) on a malformed legacy invocation.
func translatePassArguments(arguments []string) ([]string, bool) {
	flags := flag.NewFlagSet("pass", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // report errors through reportPassUsage instead

	vaultPath := flags.String("vault", "", "path to an OpenSSL-encrypted vault holding the master password")
	counter := flags.Int64("counter", -1, "LessPass counter")
	length := flags.Int("length", 0, "password length")
	symbols := flags.Bool("symbols", false, "include symbols in the generated password")
	timeout := flags.Duration("timeout", 90*time.Second, "clipboard auto-clear timeout; 0 keeps it until overwritten")
	read := flags.Bool("read", false, "print the current clipboard contents")

	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			reportPassUsage(os.Stdout, flags)
			os.Exit(0)
		}
		reportPassUsage(os.Stderr, flags)
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		return nil, false
	}

	translated := []string{"--copy", "--clear-after=" + timeout.String()}
	if *vaultPath != "" {
		translated = append(translated, "--vault", *vaultPath)
	}
	if *counter >= 0 {
		translated = append(translated, "-C", strconv.FormatInt(*counter, 10))
	}
	if *length > 0 {
		translated = append(translated, "-L", strconv.Itoa(*length))
	}
	if *symbols {
		translated = append(translated, "-s")
	}
	if *read {
		return append(translated, "--read-clipboard"), true
	}

	rest := flags.Args()
	if len(rest) != 2 {
		reportPassUsage(os.Stderr, flags)
		return nil, false
	}
	return append(translated, "--profile", rest[0], rest[1]), true
}

// reportPassUsage prints the legacy usage block, the way the original pass
// tool reported a bad command line.
func reportPassUsage(output io.Writer, flags *flag.FlagSet) {
	fmt.Fprint(output, passUsage)
	flags.SetOutput(output)
	flags.PrintDefaults()
}
