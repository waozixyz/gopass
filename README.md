# gopass

Website and browser app: [pass.waozi.xyz](https://pass.waozi.xyz/)

`gopass` is a stateless password generator: the same site, login, master
password, and settings produce the same result without a password database.
It is a new, independently written implementation of the LessPass generation
algorithm. Compatibility does not imply endorsement by or affiliation with the
LessPass project.

## Install

Prebuilt CLI and desktop downloads are available from
[pass.waozi.xyz](https://pass.waozi.xyz/#downloads).

With Go 1.24 or newer:

```sh
go install github.com/waozixyz/gopass/cmd/gopass@latest
```

Or build the command from a checkout:

```sh
go build ./cmd/gopass
```

## Use

```text
gopass [OPTIONS] SITE LOGIN [MASTER_PASSWORD]
```

The default password has 16 characters and includes lowercase letters,
uppercase letters, digits, and symbols. The default counter is 1.

Passing a master password as an argument can expose it to process inspection
and shell history. Omitting it is safer: `gopass` first checks
`LESSPASS_MASTER_PASSWORD`, then reads from the controlling terminal or console
with echo disabled. Use `--prompt` to bypass the environment and always ask. For
example:

```sh
gopass --prompt example.com alice
gopass --length 24 --counter 2 example.com alice
gopass --no-symbols --exclude '0O1Il' example.com alice
```

`--copy` sends the result to the clipboard without printing it. On Linux a
built-in pure-Go X11 clipboard is used when an X server (X.Org, Xlibre, or
XWayland) is reachable, so no external tool is needed; otherwise `gopass`
falls back to `wl-copy`, `xclip`, `xsel`, `pbcopy`, or `clip.exe` and reports
an error if none can be used. The copied password is served by a small
detached process and disappears when another application takes the clipboard;
`--clear-after 90s` releases it automatically after a delay (built-in X11
clipboard only), and `--read-clipboard` prints the current contents.

## Profiles

An optional configuration file at `$XDG_CONFIG_HOME/gopass/profiles.json`
(usually `~/.config/gopass/profiles.json`) can store named profiles — a login
plus default settings — so daily invocations shrink to one flag:

```sh
gopass --profile work example.com
```

```json
{
  "vault": "~/.vault",
  "copy": true,
  "clear_after": "90s",
  "profiles": {
    "work": {"login": "alice", "counter": 9, "length": 23, "symbols": false}
  }
}
```

Profiles hold no secrets. Command-line flags always win over profile values,
and without a configuration file gopass behaves exactly as before.

## Master password from a vault

`--vault PATH` reads the master password from a file encrypted with OpenSSL:

```sh
printf 'master password' | openssl enc -aes-256-cbc -md sha512 -a \
    -pbkdf2 -iter 100000 -salt -pass pass:... > ~/.vault
gopass --vault ~/.vault example.com alice
```

The vault passphrase is requested on the terminal with echo disabled, or read
from piped standard input, and the decrypted contents are used as the master
password — the master itself never appears in argv or the environment. The
vault path can also be set once in the profiles file.

## The `pass` shorthand

Copying or symlinking the binary as `pass` enables a compact personal mode
built on profiles:

```sh
ln -s gopass ~/bin/pass
pass work example.com
```

`pass PROFILE SITE` copies the password without printing it and clears the
clipboard after 90 seconds (`-timeout 30s` to change, `-timeout 0` to keep it
until another application takes over).

Run `gopass --help` for all flags.

## Desktop app

The optional desktop app is written in Go with Kryon and calls the same
generator package directly. It keeps the master password in process memory,
masks the master-password field, and clears copied passwords after 20 seconds
unless the clipboard has since been replaced.

```sh
git submodule update --init --recursive
make gui
./build/gopass-gui
```

## Library

```go
settings := gopass.DefaultOptions()
result, err := gopass.Generate("example.com", "alice", master, settings)
```

All inputs are interpreted as UTF-8 strings. Excluded characters are removed
from every enabled character class; generation fails if a class becomes empty,
all classes are disabled, or the requested length cannot include one character
from each class.

## License

Copyright © 2026 Waozi. Distributed under the BSD 3-Clause License. See
[`LICENSE`](LICENSE).
