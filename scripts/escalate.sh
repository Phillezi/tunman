#!/usr/bin/env bash

is_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "false"
  else
    echo "true"
  fi
}

# If not root, prepend sudo to installation commands
SUDO=""
if [ "$(is_root)" == "false" ]; then
  SUDO="sudo"
fi

$SUDO $@
