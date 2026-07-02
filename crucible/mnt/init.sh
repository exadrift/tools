#!/bin/sh

set -ex

apk update && apk add rsync openssh-client

mkdir -p /root/.ssh
cp id_ed2551* /root/.ssh
cp -R ./bundle /root/bundle
touch /root/.ssh/known_hosts
chmod 600 /root/.ssh/known_hosts
chmod 600 /root/.ssh/id_ed25519
chmod 644 /root/.ssh/id_ed25519.pub
chmod 700 /root/.ssh
chown -R root:root /root/.ssh
