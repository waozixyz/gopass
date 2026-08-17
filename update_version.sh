#!/bin/sh
set -eu

release_version=$(sed -n 's/^## \[\([0-9][0-9.]*\)\].*/\1/p' CHANGELOG.md | head -n 1)
if [ -z "$release_version" ]; then
	echo "No numeric release entry found in CHANGELOG.md" >&2
	exit 1
fi

sed -E -i "s/^const Version = \"[^\"]+\"$/const Version = \"$release_version\"/" version.go
grep -F "const Version = \"$release_version\"" version.go >/dev/null
