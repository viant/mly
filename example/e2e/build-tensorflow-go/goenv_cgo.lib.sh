
# --- Library guards ---
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  echo "This file is a Bash library; source it, do not execute." >&2
  exit 64
fi
if [[ -n "${GOENV_CGO_LIB_INCLUDED:-}" ]]; then
  return 0
fi
readonly GOENV_CGO_LIB_INCLUDED=1

GOENV_CGO_IS_MACOS=$(uname -s | grep -i 'darwin')

# --- Helpers that use `go env` safely ---
_goenv_get() { go env "$1" | tr -d '\n'; }
_goenv_set() {
  local var="$1" val="$2"
  if [[ -z "$val" ]]; then go env -u "$var" || true
  else go env -w "$var=$val"
  fi
}
_append_unique() {
  local cur="$1" tok="$2"
  if printf ' %s ' "$cur" | grep -q " $(printf '%s' "$tok" | sed 's/[].[^$\\*]/\\&/g') "; then
    printf '%s' "$cur"
  else
    [[ -z "$cur" ]] && printf '%s' "$tok" || printf '%s %s' "$cur" "$tok"
  fi
}

# --- Public API ---
goenv_cgo::push() {
  if [[ -z "$GOENV_CGO_IS_MACOS" ]]; then
    # for linux, we use LD_LIBRARY_PATH
    return 0
  fi

  # Usage: goenv_cgo::push "/usr/local/lib [/opt/homebrew/lib ...]"
  if [[ $# -lt 1 ]]; then
    echo "goenv_cgo::push: provide one or more directories" >&2
    return 2
  fi

  # Save originals only on first push
  if [[ -z "${GOENV__OLD_CGO_LDFLAGS+x}" ]]; then
    GOENV__OLD_CGO_LDFLAGS="$(_goenv_get CGO_LDFLAGS)"
  fi
  if [[ -z "${GOENV__OLD_CGO_CFLAGS+x}" ]]; then
    GOENV__OLD_CGO_CFLAGS="$(_goenv_get CGO_CFLAGS)"
  fi

  local new_ld="$GOENV__OLD_CGO_LDFLAGS"
  for dir in "$@"; do
    [[ -d "$dir" ]] && dir="$(cd "$dir" && pwd)"
    new_ld="$(_append_unique "$new_ld" "-L$dir")"
    new_ld="$(_append_unique "$new_ld" "-Wl,-rpath,$dir")"
  done

  _goenv_set CGO_LDFLAGS "$new_ld"
  # If headers are also needed:
  # local new_cf="$GOENV__OLD_CGO_CFLAGS"
  # for dir in "$@"; do new_cf="$(_append_unique "$new_cf" "-I$dir/include")"; done
  # _goenv_set CGO_CFLAGS "$new_cf"
}

goenv_cgo::pop() {
  if [[ -z "$GOENV_CGO_IS_MACOS" ]]; then
    # for linux, we use LD_LIBRARY_PATH
    return 0
  fi

  _goenv_set CGO_LDFLAGS "${GOENV__OLD_CGO_LDFLAGS-}"
  _goenv_set CGO_CFLAGS  "${GOENV__OLD_CGO_CFLAGS-}"
  unset GOENV__OLD_CGO_LDFLAGS GOENV__OLD_CGO_CFLAGS
}