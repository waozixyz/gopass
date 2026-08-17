#!/bin/sh
set -eu

release_version=$(sed -n 's/^## \[\([0-9][0-9.]*\)\].*/\1/p' CHANGELOG.md | head -n 1)
release_date=$(sed -n 's/^## \[[0-9][0-9.]*\] - \([0-9-]*\)$/\1/p' CHANGELOG.md | head -n 1)
if [ -z "$release_version" ]; then
	echo "No numeric release entry found in CHANGELOG.md" >&2
	exit 1
fi
if [ -z "$release_date" ]; then
	echo "No release date found for $release_version in CHANGELOG.md" >&2
	exit 1
fi

sed -E -i "s/^const Version = \"[^\"]+\"$/const Version = \"$release_version\"/" version.go
sed -E -i "s/versionName \"[^\"]+\"/versionName \"$release_version\"/" droid/app/build.gradle
metainfo=packaging/linux/xyz.waozi.pass.appdata.xml
if ! grep -F "<release version=\"$release_version\"" "$metainfo" >/dev/null; then
	sed -E -i "s#<releases>#<releases><release version=\"$release_version\" date=\"$release_date\"/>#" "$metainfo"
fi
grep -F "const Version = \"$release_version\"" version.go >/dev/null
grep -F "versionName \"$release_version\"" droid/app/build.gradle >/dev/null
grep -F "<release version=\"$release_version\" date=\"$release_date\"" "$metainfo" >/dev/null
