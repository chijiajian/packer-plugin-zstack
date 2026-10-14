#!/bin/sh
set -eu
mkdir -p /target/etc/ssh/sshd_config.d
printf 'PermitRootLogin yes\nPasswordAuthentication yes\n' > /target/etc/ssh/sshd_config.d/90-packer.conf
touch /target/etc/packer-installed /target/etc/packer-answer-cd-used
in-target systemctl enable ssh
in-target systemctl enable qemu-guest-agent
rm -f /target/etc/apt/apt.conf.d/00packer-cdrom
