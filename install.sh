#!/bin/sh

set -e

if [ "$1" = "" ]
then
    echo "install.sh <name>"
    exit 1
fi
RELEASE_NAME=$1

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')

LATEST_VERSION=$(curl https://raw.githubusercontent.com/exadrift/tools/refs/heads/main/${RELEASE_NAME}/VERSION)

BIN_DIR="${BIN_DIR:-${HOME}/.local/bin}"
mkdir -p ${BIN_DIR}
TARGET_PATH=${BIN_DIR}/${RELEASE_NAME}

echo "downloading ${RELEASE_NAME} version ${LATEST_VERSION}"
curl -L https://github.com/exadrift/tools/releases/download/${RELEASE_NAME}-${LATEST_VERSION}/${RELEASE_NAME}-${OS}-${ARCH} -o ${TARGET_PATH}
chmod +x ${TARGET_PATH}

echo "${RELEASE_NAME} installed at ${TARGET_PATH}"
echo "please add ${BIN_DIR} to your PATH variable if it isn't already"
