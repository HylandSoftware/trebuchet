#!/usr/bin/env bash
set -euo pipefail

export VERSION=ci
export FULL_VERSION=ci-full
if [ "${TRAVIS_PULL_REQUEST}" = "false" ] && [ "${TRAVIS_BRANCH}" = "master" ]; then
    VERSION=$(docker run -it --rm -v "$(pwd):/repo" gittools/gitversion:5.0.0-linux-debian-9-netcoreapp2.2 /repo /showvariable NuGetVersionV2 | tee /dev/tty)
    FULL_VERSION=$(docker run -it --rm -v "$(pwd):/repo" gittools/gitversion:5.0.0-linux-debian-9-netcoreapp2.2 /repo /showvariable InformationalVersion | tee /dev/tty)
fi

make

docker build -f ./Dockerfile -t "hylandsoftware/trebuchet:${VERSION%$'\r'}" .

if [ "${TRAVIS_PULL_REQUEST}" = "false" ] && [ "${TRAVIS_BRANCH}" = "master" ]; then
    echo "${DOCKER_PASSWORD}" | docker login --username "${DOCKER_USER}" --password-stdin

    docker tag "hylandsoftware/trebuchet:${VERSION%$'\r'}" "hylandsoftware/trebuchet:latest"
    docker push "hylandsoftware/trebuchet:${VERSION%$'\r'}"
    docker push "hylandsoftware/trebuchet:latest"
fi
