FROM ubuntu:16.04

RUN apt-get update \
 && apt-get -y install open-iscsi tgt \
 && apt-get clean \
 && echo GenerateName=yes > /etc/iscsi/initiatorname.iscsi

COPY tgtd-run.sh iscsid-run.sh /usr/local/bin/
