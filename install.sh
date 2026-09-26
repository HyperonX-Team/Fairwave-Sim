#!/usr/bin/env bash
#
# Fairwave installer - macOS, Linux, and Windows (Git Bash / MSYS2 / WSL).
#
# Installs the Fairwave binaries from the latest GitHub release when an
# asset exists for your platform, and otherwise builds them from source
# (which needs Go 1.22+). Idempotent: re-running upgrades in place.
#
#   curl -fsSL https://raw.githubusercontent.com/HyperonX-Team/Fairwave-Sim/main/install.sh | bash
#   bash install.sh --prefix "$HOME/.local" --add-to-path
#   bash install.sh --from-source --only fairwave,fairwave-hydra
#   bash install.sh --uninstall
#
set -euo pipefail

REPO_URL="${FAIRWAVE_REPO_URL:-https://github.com/HyperonX-Team/Fairwave-Sim}"
DEFAULT_VERSION="0.1.0"
ALL_COMPONENTS="fairwave fairwave-control fairwave-agent fairwave-hydra"

PREFIX="${FAIRWAVE_PREFIX:-$HOME/.local}"
BINDIR=""
VERSION=""
ONLY=""
FROM_SOURCE=0
UNINSTALL=0
ADD_PATH=0
DRYRUN=0

# ---- pretty output -------------------------------------------------------
log()  { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }
run()  { if [[ $DRYRUN -eq 1 ]]; then printf '  [dry-run] %s\n' "$*"; else "$@"; fi; }
have() { command -v "$1" >/dev/null 2>&1; }

usage() {
  cat <<'EOF'
Fairwave installer

Usage: install.sh [options]

Options:
  --prefix DIR     install prefix (default: ~/.local; bins go in PREFIX/bin)
  --bindir DIR     install binaries here instead of PREFIX/bin
  --version VER    install a specific release (e.g. 0.1.0); default: latest
  --only LIST      comma-separated components (default: all)
                   fairwave, fairwave-control, fairwave-agent, fairwave-hydra
  --from-source    build from source instead of downloading a release
  --add-to-path    append BINDIR to your shell profile if it is missing
  --uninstall      remove the installed binaries and exit
  --dry-run        print what would happen, change nothing
  -h, --help       show this help

Environment:
  FAIRWAVE_PREFIX     same as --prefix
  FAIRWAVE_REPO_URL   override the source/release repository URL

Windows: run this from Git Bash, MSYS2, or WSL. The binaries are installed
with a .exe suffix so they run natively.
EOF
}

# ---- argument parsing ----------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)      PREFIX="${2:?--prefix needs a value}"; shift 2 ;;
    --bindir)      BINDIR="${2:?--bindir needs a value}"; shift 2 ;;
    --version)     VERSION="${2:?--version needs a value}"; shift 2 ;;
    --only)        ONLY="${2:?--only needs a value}"; shift 2 ;;
    --from-source) FROM_SOURCE=1; shift ;;
    --add-to-path) ADD_PATH=1; shift ;;
    --uninstall)   UNINSTALL=1; shift ;;
    --dry-run)     DRYRUN=1; shift ;;
    -h|--help)     usage; exit 0 ;;
    *)             die "unknown option: $1 (try --help)" ;;
  esac
done

BINDIR="${BINDIR:-$PREFIX/bin}"

# ---- platform detection --------------------------------------------------
detect_os() {
  case "$(uname -s)" in
    Darwin)                 echo darwin ;;
    Linux)                  echo linux ;;
    MINGW*|MSYS*|CYGWIN*)   echo windows ;;
    *)                      echo "" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    armv7l|armv6l) echo arm ;;
    *)             echo "" ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
EXT=""
[[ "$OS" == windows ]] && EXT=".exe"

if [[ -z "$OS" ]]; then
  warn "unrecognised platform: $(uname -s) $(uname -m)"
  warn "will attempt a source build (needs Go)"
  FROM_SOURCE=1
fi

# ---- component selection -------------------------------------------------
normalize_component() {
  case "$1" in
    cli|fairwave|fairwave-cli)      echo fairwave ;;
    control|fairwave-control)       echo fairwave-control ;;
    agent|fairwave-agent)           echo fairwave-agent ;;
    hydra|fairwave-hydra)           echo fairwave-hydra ;;
    *) die "unknown component: $1" ;;
  esac
}

COMPONENTS=""
if [[ -n "$ONLY" ]]; then
  IFS=',' read -r -a _req <<< "$ONLY"
  for c in "${_req[@]}"; do
    c="$(printf '%s' "$c" | tr -d '[:space:]')"
    [[ -z "$c" ]] && continue
    COMPONENTS="$COMPONENTS $(normalize_component "$c")"
  done
else
  COMPONENTS="$ALL_COMPONENTS"
fi
COMPONENTS="$(printf '%s\n' "$COMPONENTS" | tr ' ' '\n' | awk 'NF && !seen[$0]++' | tr '\n' ' ')"
[[ -n "${COMPONENTS// /}" ]] || die "no components selected"

# ---- staging dir ---------------------------------------------------------
STAGE=""
SRC=""
cleanup() { [[ -n "$STAGE" && -d "$STAGE" ]] && rm -rf "$STAGE"; }
trap cleanup EXIT
STAGE="$(mktemp -d 2>/dev/null || mktemp -d -t fairwave)"

# ---- uninstall -----------------------------------------------------------
if [[ $UNINSTALL -eq 1 ]]; then
  log "uninstalling Fairwave from $BINDIR"
  for name in $COMPONENTS; do
    target="$BINDIR/${name}${EXT}"
    if [[ -e "$target" ]]; then
      run rm -f "$target"
      log "  removed $target"
    else
      log "  not installed: $target"
    fi
  done
  log "done"
  exit 0
fi

# ---- plan ----------------------------------------------------------------
METHOD="download"
[[ $FROM_SOURCE -eq 1 ]] && METHOD="source"
if [[ $DRYRUN -eq 1 ]]; then
  log "Fairwave installer (dry run)"
  log "  platform:   ${OS:-unknown}/${ARCH:-unknown}${EXT:+ (exe)}"
  log "  prefix:     $PREFIX"
  log "  bindir:     $BINDIR"
  log "  components: $COMPONENTS"
  log "  version:    ${VERSION:-<latest release, else from-source $DEFAULT_VERSION>}"
  log "  method:     $METHOD"
  exit 0
fi

# ---- fetching helpers ----------------------------------------------------
fetch() { # url dest
  if have curl; then curl -fsSL "$1" -o "$2"
  elif have wget; then wget -qO "$2" "$1"
  else die "need curl or wget to download release assets"
  fi
}

sha256_of() {
  if have sha256sum; then sha256sum "$1" | awk '{print $1}'
  elif have shasum; then shasum -a 256 "$1" | awk '{print $1}'
  else echo ""; fi
}

# ---- download path -------------------------------------------------------
download_release() {
  [[ $FROM_SOURCE -eq 1 ]] && return 1
  if [[ -z "$OS" || -z "$ARCH" ]]; then
    warn "no prebuilt asset for this platform; falling back to a source build"
    return 1
  fi

  local base
  if [[ -n "$VERSION" ]]; then
    base="$REPO_URL/releases/download/v${VERSION}"
  else
    base="$REPO_URL/releases/latest/download"
  fi

  log "fetching release assets from $base"
  local name asset url
  for name in $COMPONENTS; do
    asset="${name}-${OS}-${ARCH}"
    url="${base}/${asset}"
    if ! fetch "$url" "$STAGE/${name}${EXT}" 2>/dev/null; then
      warn "asset ${asset} not found - falling back to a source build"
      return 1
    fi
    # Verify the checksum when the release publishes one.
    if fetch "${url}.sha256" "$STAGE/${asset}.sha256" 2>/dev/null; then
      local want got
      want="$(awk '{print $1}' "$STAGE/${asset}.sha256")"
      got="$(sha256_of "$STAGE/${name}${EXT}")"
      if [[ -n "$want" && -n "$got" && "$want" != "$got" ]]; then
        die "checksum mismatch for ${asset} (want $want, got $got)"
      fi
      log "  verified ${asset}"
    else
      log "  fetched ${asset} (no checksum published)"
    fi
    chmod +x "$STAGE/${name}${EXT}" 2>/dev/null || true
  done
  return 0
}

# ---- source-build path ---------------------------------------------------
locate_source() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "")"
  if [[ -n "$here" && -f "$here/go.mod" ]] && grep -q 'Fairwave-Sim' "$here/go.mod" 2>/dev/null; then
    SRC="$here"
    return 0
  fi
  have git || die "Go and git are required to build from source (https://go.dev/dl)"
  SRC="$STAGE/src"
  local clone_args=(--depth 1)
  [[ -n "$VERSION" ]] && clone_args+=(--branch "v${VERSION}")
  log "cloning $REPO_URL (${VERSION:-default branch})"
  git clone "${clone_args[@]}" "${REPO_URL}.git" "$SRC" >/dev/null 2>&1 \
    || die "git clone failed (is the tag v${VERSION} published?)"
}

source_version() {
  if [[ -n "$VERSION" ]]; then printf '%s' "$VERSION"; return; fi
  local v=""
  if [[ -d "$SRC/.git" ]] && have git; then
    v="$(git -C "$SRC" describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || true)"
  fi
  printf '%s' "${v:-$DEFAULT_VERSION}"
}

build_from_source() {
  have go || die "Go 1.22+ is required to build from source (https://go.dev/dl)"
  locate_source
  local ver
  ver="$(source_version)"
  log "building from source at $SRC (version ${ver})"

  local ldf_control ldf_cli ldf_hydra
  ldf_control="-X github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/api.Version=${ver}"
  ldf_cli="-X github.com/HyperonX-Team/Fairwave-Sim/apps/fairwave-cli/internal/cli.Version=${ver}"
  ldf_hydra="-X main.Version=${ver}"

  local name pkg ld
  for name in $COMPONENTS; do
    case "$name" in
      fairwave)         pkg="./apps/fairwave-cli/cmd/fairwave";            ld="$ldf_cli" ;;
      fairwave-control) pkg="./core/fairwave-control/cmd/fairwave-control"; ld="$ldf_control" ;;
      fairwave-agent)   pkg="./core/fairwave-agent/cmd/fairwave-agent";     ld="$ldf_cli" ;;
      fairwave-hydra)   pkg="./core/hydra/cmd/fairwave-hydra";             ld="$ldf_hydra" ;;
    esac
    log "  go build ${name}"
    ( cd "$SRC" && CGO_ENABLED=0 go build -trimpath -ldflags "$ld" -o "$STAGE/${name}${EXT}" "$pkg" )
  done
  return 0
}

# ---- install -------------------------------------------------------------
if [[ $FROM_SOURCE -eq 1 ]]; then
  build_from_source
elif ! download_release; then
  build_from_source
fi

run mkdir -p "$BINDIR"
for name in $COMPONENTS; do
  target="$BINDIR/${name}${EXT}"
  run cp "$STAGE/${name}${EXT}" "$target"
  run chmod +x "$target"
  log "  installed $target"
done

# ---- optional PATH wiring ------------------------------------------------
if [[ $ADD_PATH -eq 1 ]]; then
  rc="$HOME/.bashrc"
  case "${SHELL:-}" in
    */zsh) rc="$HOME/.zshrc" ;;
  esac
  if [[ -f "$rc" ]] && grep -qF "$BINDIR" "$rc" 2>/dev/null; then
    log "  $BINDIR is already on PATH in $rc"
  else
    # Write a literal $PATH into the profile: it must expand when the shell
    # sources the file, not now.
    # shellcheck disable=SC2016
    printf '\n# Fairwave\nexport PATH="%s:$PATH"\n' "$BINDIR" >> "$rc"
    log "  added $BINDIR to PATH in $rc (open a new shell)"
  fi
fi

# ---- verify + summary ----------------------------------------------------
if [[ " $COMPONENTS " == *" fairwave "* && -x "$BINDIR/fairwave${EXT}" ]]; then
  ver_out="$("$BINDIR/fairwave${EXT}" version 2>/dev/null || true)"
  [[ -n "$ver_out" ]] && log "  $ver_out"
fi

log ""
log "Fairwave installed to $BINDIR"
case ":$PATH:" in
  *":$BINDIR:"*) : ;;
  *) log "Add it to PATH:  export PATH=\"$BINDIR:\$PATH\"" ;;
esac
log ""
log "Next:"
log "  fairwave version            # confirm the install"
log "  fairwave hydra status       # talk to a running control plane"
log "  make demo                   # from a checkout: local dashboard + weave"
