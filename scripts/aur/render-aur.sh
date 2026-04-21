#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <version> [--pkgrel <pkgrel>] [--sha256 <sha256>]" >&2
}

sha256_file() {
  local path="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${path}" | awk '{print $1}'
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${path}" | awk '{print $1}'
    return
  fi

  echo "No SHA-256 tool found; install sha256sum or shasum." >&2
  exit 1
}

escape_sed() {
  printf '%s' "$1" | sed -e 's/[\/&]/\\&/g'
}

version=""
pkgrel="${PKGREL:-1}"
sha256="${DLOOM_AUR_SHA256:-}"
source_sha256="${DLOOM_AUR_SOURCE_SHA256:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pkgrel)
      [[ $# -ge 2 ]] || { usage; exit 1; }
      pkgrel="$2"
      shift 2
      ;;
    --sha256)
      [[ $# -ge 2 ]] || { usage; exit 1; }
      sha256="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [[ -z "${version}" ]]; then
        version="$1"
        shift
      else
        echo "Unexpected argument: $1" >&2
        usage
        exit 1
      fi
      ;;
  esac
done

if [[ -z "${version}" ]]; then
  usage
  exit 1
fi

version="${version#v}"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
packaging_rel_dir="${AUR_PACKAGING_DIR:-packaging/aur}"
packaging_dir="${repo_root}/${packaging_rel_dir}"
template="${packaging_dir}/PKGBUILD.in"
output="${packaging_dir}/PKGBUILD"

if [[ ! -f "${template}" ]]; then
  echo "Expected ${template}" >&2
  exit 1
fi

tmp_paths=()
cleanup() {
  if [[ ${#tmp_paths[@]} -gt 0 ]]; then
    rm -f "${tmp_paths[@]}"
  fi
}
trap cleanup EXIT

case "$(basename "${packaging_dir}")" in
  aur)
    source_url="${AUR_SOURCE_URL:-https://github.com/dloomorg/dloom/archive/refs/tags/v${version}.tar.gz}"

    if [[ -z "${sha256}" ]]; then
      tmp_tarball="$(mktemp)"
      tmp_paths+=("${tmp_tarball}")
      curl -fsSL "${source_url}" -o "${tmp_tarball}"
      sha256="$(sha256_file "${tmp_tarball}")"
    fi

    sed \
      -e "s/@PKGVER@/$(escape_sed "${version}")/g" \
      -e "s/@PKGREL@/$(escape_sed "${pkgrel}")/g" \
      -e "s/@SOURCE_URL@/$(escape_sed "${source_url}")/g" \
      -e "s/@SHA256@/$(escape_sed "${sha256}")/g" \
      "${template}" > "${output}"
    ;;
  aur-bin)
    binary_url="${AUR_BINARY_URL:-https://github.com/dloomorg/dloom/releases/download/v${version}/dloom_v${version}_linux_amd64.tar.gz}"
    source_url="${AUR_SOURCE_URL:-https://github.com/dloomorg/dloom/archive/refs/tags/v${version}.tar.gz}"

    if [[ -z "${sha256}" ]]; then
      tmp_binary="$(mktemp)"
      tmp_paths+=("${tmp_binary}")
      curl -fsSL "${binary_url}" -o "${tmp_binary}"
      sha256="$(sha256_file "${tmp_binary}")"
    fi

    if [[ -z "${source_sha256}" ]]; then
      tmp_source="$(mktemp)"
      tmp_paths+=("${tmp_source}")
      curl -fsSL "${source_url}" -o "${tmp_source}"
      source_sha256="$(sha256_file "${tmp_source}")"
    fi

    sed \
      -e "s/@PKGVER@/$(escape_sed "${version}")/g" \
      -e "s/@PKGREL@/$(escape_sed "${pkgrel}")/g" \
      -e "s/@BINARY_URL@/$(escape_sed "${binary_url}")/g" \
      -e "s/@BINARY_SHA256@/$(escape_sed "${sha256}")/g" \
      -e "s/@SOURCE_URL@/$(escape_sed "${source_url}")/g" \
      -e "s/@SOURCE_SHA256@/$(escape_sed "${source_sha256}")/g" \
      "${template}" > "${output}"
    ;;
  *)
    echo "Unsupported packaging directory: ${packaging_dir}" >&2
    exit 1
    ;;
esac

echo "Rendered ${output} for ${version}-${pkgrel}"
