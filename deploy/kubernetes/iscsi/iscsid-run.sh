#!/usr/bin/env bash
set -e
/lib/open-iscsi/startup-checks.sh
exec /usr/sbin/iscsid -f
