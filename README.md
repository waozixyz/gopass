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

`--copy` sends the result through standard input to an available clipboard
program (`wl-copy`, `xclip`, `xsel`, `pbcopy`, or `clip.exe`) and does not print
the generated password. The command reports an error if no supported clipboard
tool can be used.

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
