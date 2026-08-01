#!/bin/sh

set -eu

release=1.13.7
platform=linux-x86_64
digest=ca7051b11c6bd3c9986cc64db640347594c944a177745ca462764cecd86d74e2
shared_digest=e57011e9dec09fcfda5bae267c99c96527b78fa76213a90eb464b1bb731b5f74
static_digest=2e317e26af60a6247af067e22541d949e4e5485f4ef9757c192047a682dae235
asset="tree-sitter-language-pack-go-v${release}-${platform}.tar.gz"
url="https://github.com/xberg-io/tree-sitter-language-pack/releases/download/v${release}/${asset}"
cache_root=${XDG_CACHE_HOME:-"$HOME/.cache"}
native_dir="${cache_root}/tree-sitter-language-pack/go/${release}/${platform}"
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT

if [ "$(go env GOOS)/$(go env GOARCH)" != "linux/amd64" ]; then
	printf 'code intelligence is unsupported on %s/%s\n' "$(go env GOOS)" "$(go env GOARCH)" >&2
	exit 1
fi

mkdir -p "$native_dir"
if ! printf '%s  %s\n%s  %s\n' \
	"$shared_digest" "$native_dir/libts_pack_core_ffi.so" \
	"$static_digest" "$native_dir/libts_pack_core_ffi.a" | sha256sum --check --status; then
	curl --fail --location --silent --show-error "$url" --output "$temporary/$asset"
	printf '%s  %s\n' "$digest" "$temporary/$asset" | sha256sum --check --status
	tar -xzf "$temporary/$asset" --strip-components=2 -C "$native_dir" \
		"tree-sitter-language-pack-go-v${release}-${platform}/lib/libts_pack_core_ffi.so" \
		"tree-sitter-language-pack-go-v${release}-${platform}/lib/libts_pack_core_ffi.a"
fi
printf '%s  %s\n%s  %s\n' \
	"$shared_digest" "$native_dir/libts_pack_core_ffi.so" \
	"$static_digest" "$native_dir/libts_pack_core_ffi.a" | sha256sum --check --status

go run "github.com/xberg-io/tree-sitter-language-pack/packages/go/cmd/setup@v${release}" \
	-dir internal/codeintel/languagepack
sed -i 's/RequireNativeSetup_1_13_5/RequireNativeSetup_1_13_7/' \
	internal/codeintel/languagepack/tree-sitter-language-pack_cgo_link.go
