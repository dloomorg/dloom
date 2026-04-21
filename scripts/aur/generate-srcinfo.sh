#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
packaging_dir="${repo_root}/packaging/aur"

if [[ ! -f "${packaging_dir}/PKGBUILD" ]]; then
  echo "Expected ${packaging_dir}/PKGBUILD. Run scripts/aur/render-aur.sh first." >&2
  exit 1
fi

if command -v makepkg >/dev/null 2>&1 && [[ "$(id -u)" -ne 0 ]]; then
  (
    cd "${packaging_dir}"
    makepkg --printsrcinfo > .SRCINFO
  )
  echo "Generated ${packaging_dir}/.SRCINFO"
  exit 0
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "Need either a non-root makepkg environment or Docker to generate .SRCINFO." >&2
  exit 1
fi

docker run --rm \
  --user "$(id -u):$(id -g)" \
  --volume "${repo_root}:/work" \
  --workdir /work/packaging/aur \
  archlinux:base-devel \
  bash -lc 'makepkg --printsrcinfo > .SRCINFO'

echo "Generated ${packaging_dir}/.SRCINFO"
