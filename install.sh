#!/bin/sh
# Install or update ctop from GitHub releases.
#
#   curl -fsSL https://raw.githubusercontent.com/maxeo/ctop/master/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/maxeo/ctop/master/install.sh | sh -s -- --version 0.8.9 --dir ~/.local/bin
#
# POSIX sh; needs curl or wget. Works on Linux (amd64, arm64, armv7) and macOS.
set -eu

REPO="maxeo/ctop"
VERSION=""
DIR=""
ARCH=""
UNINSTALL=0
VERIFY=1

usage() {
	cat <<EOF
Usage: install.sh [options]

  -v, --version X   install version X (e.g. 0.8.9); default: latest release
  -d, --dir DIR     install into DIR; default: where ctop already is,
                    otherwise /usr/local/bin
  -a, --arch ARCH   amd64, arm64 or arm; default: detected from uname -m
  -r, --repo O/N    GitHub repository; default: $REPO
      --no-verify   skip the sha256 check against sha256sums.txt
      --uninstall   remove ctop from DIR (or from where it is found)
  -h, --help        show this help

sudo is used only when DIR is not writable by the current user.
EOF
}

die() {
	echo "install.sh: $*" >&2
	exit 1
}

while [ $# -gt 0 ]; do
	case "$1" in
	-v | --version) VERSION="${2:?--version needs a value}"; shift 2 ;;
	-d | --dir) DIR="${2:?--dir needs a value}"; shift 2 ;;
	-a | --arch) ARCH="${2:?--arch needs a value}"; shift 2 ;;
	-r | --repo) REPO="${2:?--repo needs a value}"; shift 2 ;;
	--no-verify) VERIFY=0; shift ;;
	--uninstall) UNINSTALL=1; shift ;;
	-h | --help) usage; exit 0 ;;
	*) usage >&2; die "unknown option: $1" ;;
	esac
done

# --- helpers ----------------------------------------------------------------

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1"; }
	download() { curl -fL --progress-bar -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -qO- "$1"; }
	download() { wget -q --show-progress -O "$2" "$1" 2>/dev/null || wget -qO "$2" "$1"; }
else
	die "curl or wget is required"
fi

# run a command with sudo only when the target directory is not writable
as_owner() {
	if [ -w "$DIR" ]; then
		"$@"
	elif [ "$(id -u)" -eq 0 ]; then
		"$@"
	elif command -v sudo >/dev/null 2>&1; then
		sudo "$@"
	elif command -v doas >/dev/null 2>&1; then
		doas "$@"
	else
		die "$DIR is not writable and neither sudo nor doas is available (try --dir ~/.local/bin)"
	fi
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d' ' -f1
	else
		echo ""
	fi
}

# --- target directory -------------------------------------------------------

if [ -z "$DIR" ]; then
	existing=$(command -v ctop 2>/dev/null || true)
	case "$existing" in
	/*) DIR=$(dirname "$existing") ;;
	*) DIR=/usr/local/bin ;;
	esac
fi
case "$DIR" in
"~"/*) DIR="$HOME/${DIR#"~/"}" ;;
esac
TARGET="$DIR/ctop"

if [ "$UNINSTALL" -eq 1 ]; then
	[ -e "$TARGET" ] || die "ctop not found in $DIR"
	as_owner rm -f "$TARGET"
	echo "removed $TARGET"
	exit 0
fi

# --- platform ---------------------------------------------------------------

case "$(uname -s)" in
Linux) OS=linux ;;
Darwin) OS=darwin ;;
*) die "unsupported OS: $(uname -s) (Windows: download the .exe from the releases page)" ;;
esac

if [ -z "$ARCH" ]; then
	case "$(uname -m)" in
	x86_64 | amd64) ARCH=amd64 ;;
	aarch64 | arm64) ARCH=arm64 ;;
	armv7* | armv6*) ARCH=arm ;;
	*) die "unsupported architecture: $(uname -m) (override with --arch)" ;;
	esac
fi
case "$ARCH" in
amd64 | arm64 | arm) ;;
*) die "unsupported --arch: $ARCH (amd64, arm64 or arm)" ;;
esac
if [ "$OS" = darwin ] && [ "$ARCH" = arm ]; then
	die "no darwin/arm build"
fi

# --- version ----------------------------------------------------------------

if [ -z "$VERSION" ]; then
	VERSION=$(fetch "https://api.github.com/repos/$REPO/releases/latest" |
		sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)
	[ -n "$VERSION" ] || die "could not determine the latest release of $REPO"
fi
VERSION=${VERSION#v}

FILE="ctop-$VERSION-$OS-$ARCH"
BASE="https://github.com/$REPO/releases/download/v$VERSION"

if [ -x "$TARGET" ]; then
	current=$("$TARGET" -v 2>/dev/null | sed -n 's/^ctop version \([^,]*\).*/\1/p' || true)
	if [ "$current" = "$VERSION" ]; then
		echo "ctop $VERSION is already installed in $TARGET"
		exit 0
	fi
	echo "updating $TARGET: ${current:-unknown} -> $VERSION"
else
	echo "installing ctop $VERSION ($OS/$ARCH) into $TARGET"
fi

# --- download, verify, install ---------------------------------------------

TMP=$(mktemp -d 2>/dev/null || mktemp -d -t ctop)
trap 'rm -rf "$TMP"' EXIT INT TERM

download "$BASE/$FILE" "$TMP/ctop" || die "download failed: $BASE/$FILE"

if [ "$VERIFY" -eq 1 ]; then
	expected=$(fetch "$BASE/sha256sums.txt" 2>/dev/null | awk -v f="$FILE" '$2 == f || $2 == "*" f { print $1 }' || true)
	actual=$(sha256 "$TMP/ctop")
	if [ -z "$expected" ]; then
		echo "warning: $FILE not listed in sha256sums.txt, checksum not verified" >&2
	elif [ -z "$actual" ]; then
		echo "warning: no sha256sum/shasum available, checksum not verified" >&2
	elif [ "$expected" != "$actual" ]; then
		die "checksum mismatch for $FILE (expected $expected, got $actual)"
	fi
fi

chmod 755 "$TMP/ctop"
[ -d "$DIR" ] || mkdir -p "$DIR" 2>/dev/null || as_owner mkdir -p "$DIR"
# copy next to the target, then rename: replacing a running binary in place
# fails with "text file busy"
as_owner cp "$TMP/ctop" "$DIR/.ctop.new"
as_owner chmod 755 "$DIR/.ctop.new"
as_owner mv -f "$DIR/.ctop.new" "$TARGET"

echo "$("$TARGET" -v)"
case ":$PATH:" in
*":$DIR:"*) ;;
*) echo "note: $DIR is not in PATH" ;;
esac
