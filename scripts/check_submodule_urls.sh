#!/usr/bin/env bash
set -euo pipefail

root="${1:-.}"
cd "$root"

emit_error() {
  local file="$1"
  local message="$2"
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    printf '::error file=%s::%s\n' "$file" "$message" >&2
  else
    printf '%s: %s\n' "$file" "$message" >&2
  fi
}

check_gitmodules_file() {
  local file="$1"
  local line key url
  local status=0
  local entries=()

  [ -f "$file" ] || return 0

  if ! mapfile -t entries < <(git config --file "$file" --get-regexp '^submodule\..*\.url$'); then
    emit_error "$file" "has no submodule URL entries"
    return 1
  fi

  for line in "${entries[@]}"; do
    key="${line%% *}"
    url="${line#* }"
    case "$url" in
      https://*) ;;
      *)
        emit_error "$file" "$key uses non-HTTPS submodule URL: $url"
        status=1
        ;;
    esac
  done

  printf 'checked %s\n' "$file"
  return "$status"
}

queue=(".")

for ((i = 0; i < ${#queue[@]}; i++)); do
  repo="${queue[$i]}"
  modules_file="$repo/.gitmodules"
  [ -f "$modules_file" ] || continue

  check_gitmodules_file "$modules_file"

  mapfile -t submodule_paths < <(
    git -C "$repo" config --file .gitmodules --get-regexp '^submodule\..*\.path$' \
      | sed 's/^[^ ]* //'
  )

  for path in "${submodule_paths[@]}"; do
    if [ ! -d "$repo/$path" ]; then
      emit_error "$modules_file" "submodule path is not checked out: $path"
      exit 1
    fi
    queue+=("$repo/$path")
  done
done

echo "all discovered submodule URLs use public HTTPS"
