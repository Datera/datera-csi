#!/usr/bin/env bash
set -e
modprobe dm-multipath
service multipath-tools start
/lib/open-iscsi/startup-checks.sh
exec /usr/sbin/iscsid -f
