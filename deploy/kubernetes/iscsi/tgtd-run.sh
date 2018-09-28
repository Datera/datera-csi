#!/usr/bin/env bash
set -e
trap 'sigint' SIGINT SIGTERM

sigint()
{
    echo "starting graceful shutdown ..."
    /usr/sbin/tgtadm --op update --mode sys --name State -v offline
    /usr/sbin/tgt-admin --offline ALL
    /usr/sbin/tgt-admin --update ALL -c /dev/null -f
    /usr/sbin/tgtadm --op delete --mode system
    echo "... shutdown completed."
    exit 0
}

/usr/sbin/tgtd -f &
wait $!
