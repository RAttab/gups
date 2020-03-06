#! /usr/bin/env bash
set -o errexit \
    -o pipefail

artifact_dest="${artifact_dest:-build-metadata}"

log() {
  printf "${@}" >&2
}

cd gups-repo || exit 1

log 'sha: '
sha="$(git rev-parse | tee "../${artifact_dest}/sha")"

log 'tag: '
tag="$(git describe --exact-match --tags HEAD | tee "../${artifact_dest}/tag")"

major="$(printf "%s" "${tag}" | cut -d. -f1)"
minor="$(printf "%s" "${tag}" | cut -d. -f2)"

log 'additionnal_tags: '
echo "${major} ${major}.${minor}" | tee "../${artifact_dest}/additionnal_tags"

sha="$(git rev-parse HEAD)"

log "labels.json: \n"
tee "../${artifact_dest}/labels.json" <<EOF
{
  "BUILT_FROM_REF": "${sha}"
}
EOF
