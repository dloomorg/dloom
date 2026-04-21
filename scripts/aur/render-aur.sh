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

version=""
pkgrel="${PKGREL:-1}"
sha256="${DLOOM_AUR_SHA256:-}"

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
packaging_dir="${repo_root}/packaging/aur"
template="${packaging_dir}/PKGBUILD.in"
output="${packaging_dir}/PKGBUILD"
source_url="https://github.com/dloomorg/dloom/archive/refs/tags/v${version}.tar.gz"

if [[ -z "${sha256}" ]]; then
  tmp_tarball="$(mktemp)"
  trap 'rm -f "${tmp_tarball}"' EXIT
  curl -fsSL "${source_url}" -o "${tmp_tarball}"
  sha256="$(sha256_file "${tmp_tarball}")"
fi

sed \
  -e "s/@PKGVER@/${version}/g" \
  -e "s/@PKGREL@/${pkgrel}/g" \
  -e "s/@SHA256@/${sha256}/g" \
  "${template}" > "${output}"

echo "Rendered ${output} for ${version}-${pkgrel}"
