# Changelog

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
