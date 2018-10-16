FROM alpine:3.7

RUN apk add --no-cache ca-certificates \
                       logrotate \
                       e2fsprogs \
                       xfsprogs \
                       btrfs-progs \
                       zfs \
                       mkinitfs \
                       open-iscsi


ADD ../../assets/driver-logrotate /etc/logrotate.d/
ADD dat-csi-plugin /bin/

ENTRYPOINT ["/bin/dat-csi-plugin | tee /var/log/driver.log"]
