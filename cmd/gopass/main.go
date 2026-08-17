package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	password "github.com/waozixyz/gopass"
)

const environmentVariable = "LESSPASS_MASTER_PASSWORD"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gopass: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	options := password.DefaultOptions()
	flags := flag.NewFlagSet("gopass", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printHelp(stderr) }

	flags.IntVar(&options.Length, "L", options.Length, "password length")
	flags.IntVar(&options.Length, "length", options.Length, "password length")
	flags.Uint64Var(&options.Counter, "C", options.Counter, "generation counter")
	flags.Uint64Var(&options.Counter, "counter", options.Counter, "generation counter")
	flags.BoolVar(&options.Lowercase, "l", options.Lowercase, "include lowercase letters")
	flags.BoolVar(&options.Lowercase, "lowercase", options.Lowercase, "include lowercase letters")
	flags.BoolVar(&options.Uppercase, "u", options.Uppercase, "include uppercase letters")
	flags.BoolVar(&options.Uppercase, "uppercase", options.Uppercase, "include uppercase letters")
	flags.BoolVar(&options.Digits, "d", options.Digits, "include digits")
	flags.BoolVar(&options.Digits, "digits", options.Digits, "include digits")
	flags.BoolVar(&options.Symbols, "s", options.Symbols, "include symbols")
	flags.BoolVar(&options.Symbols, "symbols", options.Symbols, "include symbols")
	var noLowercase, noUppercase, noDigits, noSymbols bool
	flags.BoolVar(&noLowercase, "no-lowercase", false, "exclude lowercase letters")
	flags.BoolVar(&noUppercase, "no-uppercase", false, "exclude uppercase letters")
	flags.BoolVar(&noDigits, "no-digits", false, "exclude digits")
	flags.BoolVar(&noSymbols, "no-symbols", false, "exclude symbols")
	flags.StringVar(&options.Exclude, "exclude", "", "characters to exclude")

	var forcePrompt, copyPassword, showVersion bool
	flags.BoolVar(&forcePrompt, "p", false, "prompt for the master password")
	flags.BoolVar(&forcePrompt, "prompt", false, "prompt for the master password")
	flags.BoolVar(&copyPassword, "c", false, "copy the password to the clipboard")
	flags.BoolVar(&copyPassword, "copy", false, "copy the password to the clipboard")
	flags.BoolVar(&showVersion, "v", false, "show version")
	flags.BoolVar(&showVersion, "version", false, "show version")

	if err := flags.Parse(normalizeArguments(arguments)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if noLowercase {
		options.Lowercase = false
	}
	if noUppercase {
		options.Uppercase = false
	}
	if noDigits {
		options.Digits = false
	}
	if noSymbols {
		options.Symbols = false
	}
	if showVersion {
		fmt.Fprintf(stdout, "gopass %s\n", password.Version)
		return nil
	}

	positional := flags.Args()
	if len(positional) < 2 || len(positional) > 3 {
		printHelp(stderr)
		return errors.New("expected SITE LOGIN and an optional MASTER_PASSWORD")
	}
	if forcePrompt && len(positional) == 3 {
		return errors.New("MASTER_PASSWORD cannot be supplied together with --prompt")
	}

	master, err := masterPassword(positional, forcePrompt)
	if err != nil {
		return err
	}
	generated, err := password.Generate(positional[0], positional[1], master, options)
	if err != nil {
		return err
	}
	if copyPassword {
		if err := copyToClipboard(generated); err != nil {
			return err
		}
		fmt.Fprintln(stderr, "Password copied to the clipboard.")
		return nil
	}
	fmt.Fprintln(stdout, generated)
	return nil
}

// normalizeArguments lets callers place options before or after SITE and
// LOGIN. The standard flag package stops at the first positional argument,
// while command wrappers naturally append profile options after the site.
func normalizeArguments(arguments []string) []string {
	valueFlags := map[string]bool{
		"-L": true, "--length": true,
		"-C": true, "--counter": true,
		"--exclude": true,
	}
	options := make([]string, 0, len(arguments))
	positional := make([]string, 0, 3)

	for i := 0; i < len(arguments); i++ {
		argument := arguments[i]
		if argument == "--" {
			positional = append(positional, arguments[i+1:]...)
			break
		}
		if strings.HasPrefix(argument, "-") && argument != "-" {
			options = append(options, argument)
			if valueFlags[argument] && i+1 < len(arguments) {
				i++
				options = append(options, arguments[i])
			}
			continue
		}
		positional = append(positional, argument)
	}
	return append(options, positional...)
}

func masterPassword(positional []string, forcePrompt bool) (string, error) {
	if !forcePrompt && len(positional) == 3 {
		return positional[2], nil
	}
	if !forcePrompt {
		if value, exists := os.LookupEnv(environmentVariable); exists {
			return value, nil
		}
	}
	return readPassword("Master password: ")
}

func printHelp(output io.Writer) {
	fmt.Fprintln(output, `Usage: gopass [OPTIONS] SITE LOGIN [MASTER_PASSWORD]

Derive a site-specific password. If MASTER_PASSWORD is omitted, gopass reads
LESSPASS_MASTER_PASSWORD or asks for it on the controlling terminal.

Options:
  -L, --length N          password length (default 16)
  -C, --counter N         generation counter (default 1)
  -l, --lowercase         include lowercase letters
  -u, --uppercase         include uppercase letters
  -d, --digits            include digits
  -s, --symbols           include symbols
      --no-lowercase      exclude lowercase letters
      --no-uppercase      exclude uppercase letters
      --no-digits         exclude digits
      --no-symbols        exclude symbols
      --exclude CHARS     remove characters from all enabled classes
  -p, --prompt            always ask for the master password
  -c, --copy              copy result without printing it
  -v, --version           show version
  -h, --help              show this help`)
}
