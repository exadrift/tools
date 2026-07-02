#!/bin/sh

if [ ! -z $SSH_AUTH_SOCK ]
then
  echo "setting ssh agent socket"
  ssh-agent -a $SSH_AUTH_SOCK
else
  ssh-agent
fi

if [ ! -z $COMPLETION_FILE ]
then
  echo "writing completion file"
  touch $COMPLETION_FILE
fi

sleep infinity
