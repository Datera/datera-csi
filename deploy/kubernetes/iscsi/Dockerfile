FROM ubuntu:16.04

RUN apt-get update \
 && apt-get -y install open-iscsi \
                       multipath-tools \
 && echo GenerateName=yes > /etc/iscsi/initiatorname.iscsi

COPY assets/multipath.conf /etc/multipath.conf
COPY cmd/dat-csi-plugin/iscsi-recv /usr/local/bin/
COPY deploy/kubernetes/iscsi/iscsid-run.sh /usr/local/bin/
