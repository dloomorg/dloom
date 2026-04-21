#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
packaging_rel_dir="${AUR_PACKAGING_DIR:-packaging/aur}"
source_dir="${repo_root}/${packaging_rel_dir}"
pkgname="${AUR_PKGNAME:-dloom}"
aur_repo="${AUR_REPO:-ssh://aur@aur.archlinux.org/${pkgname}.git}"
aur_branch="${AUR_BRANCH:-master}"
commit_name="${AUR_COMMIT_NAME:-GitHub Actions}"
commit_email="${AUR_COMMIT_EMAIL:-41898282+github-actions[bot]@users.noreply.github.com}"

for required_file in PKGBUILD .SRCINFO LICENSE; do
  if [[ ! -f "${source_dir}/${required_file}" ]]; then
    echo "Expected ${source_dir}/${required_file}. Generate the package artifacts first." >&2
    exit 1
  fi
done

git_ssh_command="ssh"
if [[ -n "${AUR_SSH_KEY_PATH:-}" ]]; then
  git_ssh_command="ssh -i ${AUR_SSH_KEY_PATH} -o IdentitiesOnly=yes"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT
clone_dir="${tmp_dir}/${pkgname}"

GIT_SSH_COMMAND="${git_ssh_command}" git clone "${aur_repo}" "${clone_dir}"

git -C "${clone_dir}" config user.name "${commit_name}"
git -C "${clone_dir}" config user.email "${commit_email}"
git -C "${clone_dir}" config commit.gpgsign false

rsync -a --delete \
  --exclude '.git/' \
  --exclude 'PKGBUILD.in' \
  --exclude '.DS_Store' \
  "${source_dir}/" "${clone_dir}/"

if [[ -z "$(git -C "${clone_dir}" status --short)" ]]; then
  echo "AUR repo is already up to date."
  exit 0
fi

pkgver="$(sed -n 's/^pkgver=//p' "${source_dir}/PKGBUILD")"
pkgrel="$(sed -n 's/^pkgrel=//p' "${source_dir}/PKGBUILD")"

git -C "${clone_dir}" add -A
git -C "${clone_dir}" commit -m "Update ${pkgname} to ${pkgver}-${pkgrel}"
GIT_SSH_COMMAND="${git_ssh_command}" git -C "${clone_dir}" push origin "HEAD:${aur_branch}"

echo "Pushed ${pkgname} ${pkgver}-${pkgrel} to ${aur_repo}"
