#!/bin/sh
# privacy_scan.sh — refuse to ship private terms.
#
# Scans a git tree (contents and file paths) or a directory for the
# fixed-string terms listed in TERMS_FILE. The terms file lives outside
# every repository — ~/.config/git/private-terms.txt for the local
# pre-push hook, the PRIVATE_TERMS Actions secret in CI — so this script,
# its workflow, and its self-test never contain or reveal the private
# values. Reports redact every term before printing.
#
# Usage: privacy_scan.sh TERMS_FILE TREE-ISH|DIRECTORY [TREE-ISH...]
# Exit:  0 clean, 1 a term was found, 2 bad usage / unusable terms file.

set -u

if [ "$#" -lt 2 ]; then
	echo "usage: $0 TERMS_FILE TREE-ISH|DIRECTORY..." >&2
	exit 2
fi

terms_file=$1
shift

if [ ! -f "$terms_file" ]; then
	echo "privacy: terms file $terms_file not found" >&2
	exit 2
fi

# Drop comments and blank lines; require at least one term.
tmp_terms=$(mktemp) || exit 2
trap 'rm -f "$tmp_terms"' EXIT
chmod 600 "$tmp_terms"
sed -e 's/#.*$//' -e '/^[[:space:]]*$/d' "$terms_file" > "$tmp_terms"
if [ ! -s "$tmp_terms" ]; then
	echo "privacy: terms file $terms_file has no terms" >&2
	exit 2
fi

redact() {
	awk -v terms_file="$tmp_terms" '
		BEGIN { while ((getline t < terms_file) > 0) terms[++nt] = tolower(t) }
		{
			line = $0
			for (i = 1; i <= nt; i++) {
				low = tolower(line)
				while ((p = index(low, terms[i])) > 0) {
					line = substr(line, 1, p - 1) "***" substr(line, p + length(terms[i]))
					low = tolower(line)
				}
			}
			print line
		}'
}

found=0
for target in "$@"; do
	if git rev-parse -q --verify "$target^{tree}" >/dev/null 2>&1; then
		# A git tree: search contents and file paths at that revision.
		hits=$(git grep -I -iF -f "$tmp_terms" "$target" -- . 2>/dev/null || true)
		paths=$(git ls-tree -r --name-only "$target" 2>/dev/null | grep -iF -f "$tmp_terms" || true)
	elif [ -d "$target" ]; then
		# A directory: search contents and file paths on disk.
		hits=$(grep -rI -iF -f "$tmp_terms" --exclude-dir=.git "$target" 2>/dev/null || true)
		paths=$(cd "$target" && find . -name .git -prune -o -print 2>/dev/null | grep -iF -f "$tmp_terms" || true)
	else
		echo "privacy: cannot scan $target (not a tree-ish or directory)" >&2
		exit 2
	fi
	if [ -n "$hits" ]; then
		found=1
		# git grep and grep -r output already carries the tree/path prefix.
		printf '%s\n' "$hits" | redact >&2
	fi
	if [ -n "$paths" ]; then
		found=1
		printf '%s\n' "$paths" | sed 's|^|path name match: |' | redact >&2
	fi
done

if [ "$found" -ne 0 ]; then
	echo "privacy: PRIVATE DATA DETECTED — refusing to ship" >&2
	exit 1
fi
exit 0
