#! /usr/bin/env sh
set -o errexit

# If the current commit is tagged and the git status is clean
# ie. It's a release.
if git describe --exact-match HEAD && git diff --quiet; then
  tag="$(git describe --exact-match HEAD)"
  major="$(echo "${tag}" | cut -d. -f1)"
  minor="$(echo "${tag}" | cut -d. -f2)"

  docker build \
    --build-arg "BUILT_FROM_REF=$(git rev-parse HEAD)" \
    -t registry.hub.docker.com/rattab/gups:latest \
    -t "registry.hub.docker.com/rattab/gups:${tag}" \
    -t "registry.hub.docker.com/rattab/gups:${major}" \
    -t "registry.hub.docker.com/rattab/gups:${major}.${minor}" \
    .
else
  docker build \
    --build-arg "BUILT_FROM_REF=$(git rev-parse HEAD)" \
    -t "registry.hub.docker.com/rattab/gups:dev-$(date +%s)" \
    .
fi
