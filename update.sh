#!/bin/sh
# Update the installed coralline statusline from this repository:
# pull -> test -> build (to a temp file, so a failed build never touches the
# installed binary) -> deploy -> sync themes. Any failure aborts with a
# non-zero exit code and leaves the current installation untouched.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
install_dir="$HOME/.claude/coralline"
installed_bin="$install_dir/coralline"

if [ ! -d "$install_dir" ]; then
    echo "Install dir not found: $install_dir — run configure.sh for first-time setup." >&2
    exit 1
fi

cd "$repo_root"

git pull
go test ./...

tmp_bin="$install_dir/coralline.new"
go build -o "$tmp_bin" ./cmd/coralline
mv -f "$tmp_bin" "$installed_bin"

# Sync themes that differ (or are new) from repo to install dir.
mkdir -p "$install_dir/themes"
for src in "$repo_root"/themes/*.conf; do
    dst="$install_dir/themes/$(basename "$src")"
    if [ ! -f "$dst" ] || ! cmp -s "$src" "$dst"; then
        cp "$src" "$dst"
        echo "theme updated: $(basename "$src")"
    fi
done

echo "coralline updated."
