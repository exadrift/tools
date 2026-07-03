#!/bin/sh

set -e

CONTAINER_NAME=temp-exadrift

if [ "$(whoami)" != "root" ]
then
    echo "this installer requires root privileges in order to place files in the /usr/local/bin path"
    sudo echo "prompted for sudo caching"
fi

if [ "$1" = "" ]
then
    echo "install.sh <name> <version>"
    exit 1
fi
IMAGE_NAME=$1

if [ "$2" = "" ]
then
    echo "install.sh <name> <version>"
    exit 1
fi
IMAGE_TAG=$2

DOCKER_LOC=$(which docker)
if [ "${DOCKER_LOC}" = "" ]
then
    echo "you must have docker installed in order to user the installer"
    exit 1
fi

TARGET_PATH=/usr/local/bin/${IMAGE_NAME}

docker rm -f ${CONTAINER_NAME}
docker container create --name ${CONTAINER_NAME} exadrift/${IMAGE_NAME}:${IMAGE_TAG}
docker container cp ${CONTAINER_NAME}:/${IMAGE_NAME} ${TARGET_PATH}
chmod +x /usr/local/bin/${IMAGE_NAME}
docker rm -f ${CONTAINER_NAME}
echo "${IMAGE_NAME} installed at ${TARGET_PATH}"
