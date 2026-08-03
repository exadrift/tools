#!/bin/sh

set -e

if [ "$(id -u)" != "0" ]
then
    echo "this installer requires root privileges in order to place files in the /usr/local/bin path"
    exit 1
fi

if [ "$1" = "" ]
then
    echo "install.sh <name>"
    exit 1
fi
RELEASE_NAME=$1

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')

LATEST_VERSION=$(curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/${RELEASE_NAME}/VERSION)
TARGET_PATH=/usr/local/bin/${RELEASE_NAME}

curl -L https://github.com/exadrift/tools/releases/download/kubex-${LATEST_VERSION}/${RELEASE_NAME}-${OS}-${ARCH} -o ${TARGET_PATH}
chmod +x ${TARGET_PATH}

echo "${RELEASE_NAME} installed at ${TARGET_PATH}"
