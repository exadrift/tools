#!/bin/sh

set -e

CONTAINER_NAME=temp-exadrift

if [ "$(id -u)" != "0" ]
then
    echo "this installer requires root privileges in order to place files in the /usr/local/bin path"
    exit 1
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
DOCKER_IMAGE=exadrift/${IMAGE_NAME}:${IMAGE_TAG}

docker rm -f ${CONTAINER_NAME} > /dev/null 2>&1
docker pull ${DOCKER_IMAGE}
docker container create --name ${CONTAINER_NAME} ${DOCKER_IMAGE}
docker container cp ${CONTAINER_NAME}:/${IMAGE_NAME} ${TARGET_PATH}
chmod +x /usr/local/bin/${IMAGE_NAME}
docker rm -f ${CONTAINER_NAME}
echo "${IMAGE_NAME} installed at ${TARGET_PATH}"
