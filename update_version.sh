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

# Android versionName and versionCode: bump the code only when preparing a
# different release, so rerunning this script for the same version is
# idempotent.
gradle_file=droid/app/build.gradle
current_name=$(sed -n 's/.*versionName "\([^"]*\)".*/\1/p' "$gradle_file" | head -n 1)
current_code=$(sed -n 's/^[[:space:]]*versionCode \([0-9][0-9]*\).*/\1/p' "$gradle_file" | head -n 1)
if [ -z "$current_code" ]; then
	echo "Could not read versionCode from $gradle_file" >&2
	exit 1
fi
if [ "$current_name" = "$release_version" ]; then
	release_code=$current_code
else
	release_code=$((current_code + 1))
fi
sed -E -i "s/versionName \"[^\"]+\"/versionName \"$release_version\"/" "$gradle_file"
sed -E -i "s/^([[:space:]]*)versionCode [0-9]+/\1versionCode $release_code/" "$gradle_file"
metainfo=packaging/linux/xyz.waozi.pass.appdata.xml
if ! grep -F "<release version=\"$release_version\"" "$metainfo" >/dev/null; then
	sed -E -i "s#<releases>#<releases><release version=\"$release_version\" date=\"$release_date\"/>#" "$metainfo"
fi
grep -F "const Version = \"$release_version\"" version.go >/dev/null
grep -F "versionName \"$release_version\"" "$gradle_file" >/dev/null
grep -E "^[[:space:]]*versionCode $release_code$" "$gradle_file" >/dev/null
grep -F "<release version=\"$release_version\" date=\"$release_date\"" "$metainfo" >/dev/null
