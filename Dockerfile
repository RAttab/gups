FROM registry.hub.docker.com/library/golang:1.14.0-alpine3.11 AS build

COPY . /build

RUN set -o errexit; \
  apk add -U alpine-sdk;

RUN set -o errexit; \
  cd /build || exit 1; \
  ./ci/scripts/build-code.sh

FROM registry.hub.docker.com/library/alpine:3.11
ARG BUILT_FROM_REF=latest

COPY --from=build /build/gups /usr/local/bin/gups

ENV CONFIG=/etc/gups/config.json
LABEL maintainer=rattab@gmail.com BUILT_FROM_REF=${BUILT_FROM_REF}
ENTRYPOINT [ "/usr/local/bin/gups" ]
