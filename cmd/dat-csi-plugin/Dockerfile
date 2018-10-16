FROM alpine:3.7

RUN apk add --no-cache ca-certificates \
                       logrotate \
                       e2fsprogs \
                       xfsprogs \
                       btrfs-progs \
                       zfs \
                       mkinitfs \
                       open-iscsi


ADD assets/driver-logrotate /etc/logrotate.d/
ADD cmd/dat-csi-plugin/dat-csi-plugin /bin/

ENTRYPOINT [ "/bin/sh", "-c", "/bin/dat-csi-plugin 2>&1 | tee /var/log/driver.log" ]
