#!/bin/sh

if [ ! -f /etc/config/hostkeys/ssh_host_rsa_key ]
then
  # prepare host keys
  mkdir -p /etc/config/hostkeys
  mkdir -p /tmp/etc/ssh
  ssh-keygen -A -f /tmp
  mv /tmp/etc/ssh/* /etc/config/hostkeys
  rm -rf /tmp/etc
else
  echo "host keys available, no need to create new ones"
fi

# put the stored host keys in the ssh directory
cp /etc/config/hostkeys/* /etc/ssh/

if [ -f /etc/config/config.sh ]
then
  echo "running user provided configuration script"
  /etc/config/config.sh
else
  echo "no user configuration script provided"
fi

echo "starting sshd and waiting for connections"
/usr/sbin/sshd -D -e