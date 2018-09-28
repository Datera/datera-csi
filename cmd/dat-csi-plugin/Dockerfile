FROM alpine:3.7

RUN apk add --no-cache ca-certificates \
                       e2fsprogs \
                       xfsprogs \
                       btrfs-progs \
                       zfs \
                       mkinitfs \
                       open-iscsi


ADD dat-csi-plugin /bin/

ENTRYPOINT ["/bin/dat-csi-plugin"]
