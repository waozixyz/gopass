package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	password "github.com/waozixyz/pass"
	"github.com/waozixyz/pass/vault"
)

const environmentVariable = "LESSPASS_MASTER_PASSWORD"

// promptVaultPassphrase asks for the vault passphrase on the terminal, or
// reads it from standard input when stdin is piped ("echo pw | pass ...").
func promptVaultPassphrase() (string, error) {
	if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice == 0 {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	return readPassword("Vault passphrase: ")
}

// readVaultPassphrase is the hook tests replace to decrypt a synthetic vault.
var readVaultPassphrase = promptVaultPassphrase

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "pass: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	options := password.DefaultOptions()
	flags := flag.NewFlagSet("pass", flag.ContinueOnError)
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

	var forcePrompt, copyPassword, noCopy, showVersion, readClipboard, serveClipboard bool
	flags.BoolVar(&forcePrompt, "p", false, "prompt for the master password")
	flags.BoolVar(&forcePrompt, "prompt", false, "prompt for the master password")
	flags.BoolVar(&copyPassword, "c", false, "copy the password to the clipboard")
	flags.BoolVar(&copyPassword, "copy", false, "copy the password to the clipboard")
	flags.BoolVar(&noCopy, "no-copy", false, "print instead of copying when the profile says to copy")
	flags.BoolVar(&showVersion, "v", false, "show version")
	flags.BoolVar(&showVersion, "version", false, "show version")

	var profileName, vaultFlag string
	var clearAfter time.Duration
	flags.StringVar(&profileName, "P", "", "use a named profile from the configuration file")
	flags.StringVar(&profileName, "profile", "", "use a named profile from the configuration file")
	flags.StringVar(&vaultFlag, "vault", "", "read the master password from an OpenSSL-encrypted vault")
	flags.DurationVar(&clearAfter, "clear-after", 0, "clear the clipboard after this long; 0 keeps it until replaced (built-in X11 clipboard only)")
	flags.BoolVar(&readClipboard, "read-clipboard", false, "print the current clipboard contents")
	flags.BoolVar(&serveClipboard, "serve-clipboard", false, "") // internal: detached clipboard daemon mode

	if err := flags.Parse(normalizeArguments(arguments)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	explicit := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
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
		fmt.Fprintf(stdout, "pass %s\n", password.Version)
		return nil
	}
	if serveClipboard {
		return runClipboardDaemon(clearAfter)
	}
	if readClipboard {
		text, err := readClipboardSelection()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, text)
		return nil
	}

	positional := flags.Args()
	if profileName == "" && len(positional) == 2 && config != nil {
		if _, ok := config.Profiles[positional[0]]; ok {
			profileName = positional[0]
			positional = positional[1:]
		}
	}

	var profile Profile
	if profileName != "" {
		if config == nil {
			return fmt.Errorf("unknown profile %q: no configuration file at %s", profileName, configPath())
		}
		found, ok := config.Profiles[profileName]
		if !ok {
			return fmt.Errorf("unknown profile %q", profileName)
		}
		if found.Login == "" {
			return fmt.Errorf("profile %q has no login", profileName)
		}
		profile = found
		profile.apply(&options, explicit)
	}

	if profileName != "" {
		if len(positional) != 1 {
			printHelp(stderr)
			return errors.New("expected SITE with --profile")
		}
	} else if len(positional) < 2 || len(positional) > 3 {
		printHelp(stderr)
		return errors.New("expected SITE LOGIN and an optional MASTER_PASSWORD")
	}
	if forcePrompt && len(positional) == 3 {
		return errors.New("MASTER_PASSWORD cannot be supplied together with --prompt")
	}

	site, login := positional[0], ""
	if profileName != "" {
		login = profile.Login
	} else {
		login = positional[1]
	}

	// Explicit command-line password sources win. A configured vault beats
	// LESSPASS_MASTER_PASSWORD because profiles are meant to be complete
	// saved identities, including the master source.
	vaultPath := vaultFlag
	if vaultPath == "" {
		if !forcePrompt && len(positional) < 3 {
			switch {
			case profile.Vault != "":
				vaultPath = expandHome(profile.Vault)
			case config != nil && config.Vault != "":
				vaultPath = expandHome(config.Vault)
			}
		}
	}
	if !noCopy && !anySet(explicit, "c", "copy") {
		switch {
		case profile.Copy != nil:
			copyPassword = *profile.Copy
		case config != nil && config.Copy != nil:
			copyPassword = *config.Copy
		}
	}
	if noCopy {
		copyPassword = false
	}
	if !anySet(explicit, "clear-after") {
		setting := profile.ClearAfter
		if setting == "" && config != nil {
			setting = config.ClearAfter
		}
		if setting != "" {
			duration, err := time.ParseDuration(setting)
			if err != nil {
				return fmt.Errorf("clear_after: %w", err)
			}
			clearAfter = duration
		}
	}

	master, err := masterPassword(positional, forcePrompt, vaultPath)
	if err != nil {
		return err
	}
	generated, err := password.Generate(site, login, master, options)
	if err != nil {
		return err
	}
	if copyPassword {
		if err := copyToClipboard(generated, clearAfter); err != nil {
			return err
		}
		if clearAfter > 0 {
			fmt.Fprintf(stderr, "Password copied to the clipboard (auto-clears in %s).\n", clearAfter)
		} else {
			fmt.Fprintln(stderr, "Password copied to the clipboard.")
		}
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
		"-P":        true, "--profile": true,
		"--vault": true, "--clear-after": true,
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

func masterPassword(positional []string, forcePrompt bool, vaultPath string) (string, error) {
	if vaultPath != "" {
		if forcePrompt {
			return "", errors.New("--vault cannot be combined with --prompt")
		}
		if len(positional) == 3 {
			return "", errors.New("--vault cannot be combined with MASTER_PASSWORD")
		}
		passphrase, err := readVaultPassphrase()
		if err != nil {
			return "", err
		}
		return vault.Decrypt(vaultPath, passphrase)
	}
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
	fmt.Fprintln(output, `Usage: pass [OPTIONS] SITE LOGIN [MASTER_PASSWORD]
       pass [OPTIONS] PROFILE SITE

Derive a site-specific password. If MASTER_PASSWORD is omitted, pass reads
LESSPASS_MASTER_PASSWORD or asks for it on the controlling terminal. With
--vault it instead decrypts an OpenSSL-encrypted vault and uses its contents
as the master password. A named profile from the configuration file can be
used either as "pass PROFILE SITE" or with --profile; it supplies the login
and default settings.

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
      --no-copy           print instead of copying when the profile says to copy
  -P, --profile NAME      use a named profile from the configuration file
      --vault PATH        read the master password from an OpenSSL-encrypted vault
      --clear-after DUR   clear the clipboard after DUR; 0 keeps it until replaced
      --read-clipboard    print the current clipboard contents
  -v, --version           show version
  -h, --help              show this help`)
}
