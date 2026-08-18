#!/bin/sh
set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
out_dir="$root_dir/build/site"
goroot=$(go env GOROOT)

rm -rf "$out_dir"
mkdir -p "$out_dir/app"
cp "$root_dir/web/site/index.html" "$out_dir/index.html"
cp "$root_dir/web/site/styles.css" "$out_dir/styles.css"
cp "$root_dir/web/site/home.js" "$out_dir/home.js"
cp "$root_dir/web/site/app/index.html" "$out_dir/app/index.html"
cp "$root_dir/web/site/app/app.js" "$out_dir/app/app.js"
cp "$root_dir/web/site/CNAME" "$out_dir/CNAME"
cp "$root_dir/web/site/robots.txt" "$out_dir/robots.txt"
cp "$root_dir/web/site/sitemap.xml" "$out_dir/sitemap.xml"
cp "$root_dir/web/site/_redirects" "$out_dir/_redirects"
cp "$root_dir/web/site/manifest.webmanifest" "$out_dir/manifest.webmanifest"
cp -R "$root_dir/web/site/icons" "$out_dir/icons"
cp -R "$root_dir/web/site/app/icons" "$out_dir/app/icons"
cp "$root_dir/web/site/app/sw.js" "$out_dir/app/sw.js"
cp "$goroot/lib/wasm/wasm_exec.js" "$out_dir/app/wasm_exec.js" 2>/dev/null || \
	cp "$goroot/misc/wasm/wasm_exec.js" "$out_dir/app/wasm_exec.js"
(cd "$root_dir/web/wasm" && GOOS=js GOARCH=wasm go build -o "$out_dir/app/gopass.wasm" .)

test -s "$out_dir/app/gopass.wasm"
test -s "$out_dir/home.js"
printf 'built site at %s\n' "$out_dir"
