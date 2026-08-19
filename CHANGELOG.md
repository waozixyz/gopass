# Changelog

## [0.2.1] - 2026-08-19

- The desktop GUI now checks for new releases and can update itself: AppImage
  builds download the checksum-verified update and restart straight into it,
  while tarball and Debian package installs get a link to the release page.

## [0.2.0] - 2026-08-18

- Copy passwords through a built-in pure-Go X11 clipboard on Linux, with no
  external clipboard tool required; other systems keep using wl-copy, xclip,
  xsel, pbcopy, or clip.exe.
- Clear copied passwords automatically after a delay with `--clear-after`;
  by default they stay until another application takes the clipboard.
- Read the master password from an OpenSSL-encrypted vault with `--vault`;
  the vault passphrase is asked for on the terminal or read from piped input,
  so the master password never appears in argv or the environment.
- Save named profiles (a login plus default settings) in an optional
  configuration file and generate with `--profile`.
- Print the current clipboard contents with `--read-clipboard`.
- Install the binary as `pass` to use the compact `pass PROFILE SITE` form,
  which copies without printing.

## [0.1.3] - 2026-08-17

- Generate passwords directly on the website's home page: the full browser
  generator now runs in the front-page card, with no separate page to open.
- The browser generator loads only when you start using it, keeping the
  front page fast for everyone else.
- The standalone web app page remains available with the same generator.

## [0.1.2] - 2026-08-17

- Make keyboard navigation reliable: click a field, then use Tab and Shift+Tab
  to move through the form.
- Keep one gopass window open by default; launching it again replaces the
  existing window.
- Keep long editing sessions responsive and stable.

## [0.1.1] - 2026-08-17

- Keep text entry responsive and stable during prolonged use.
- Render password symbols and emoji correctly without repeatedly rebuilding fonts.
- Support selecting and replacing text in password and regular text fields.
- Use `xyz.waozi.pass` consistently as the desktop and Android application ID.

## [0.1.0] - 2026-08-17

- Generate deterministic, LessPass-compatible passwords from site, login, and
  master password inputs.
- Configure length, counter, character classes, and excluded characters.
- Read a master password without terminal echo or from an environment variable.
- Copy generated passwords with common desktop clipboard tools.
- Use the optional Kryon desktop interface to generate and briefly copy
  passwords without passing the master password to another process.
- Generate passwords locally in a browser at pass.waozi.xyz.
- Download separate CLI and desktop builds for supported processor
  architectures from the project website.
