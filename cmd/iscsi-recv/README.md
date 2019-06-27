# iSCSI Recv

## Purpose

This binary is designed to act as an implementation of a reciver for the
iscsi-rpc.proto file.  This is one of two parts (the other being iscsi-send)
that exist to allow for using the host iSCSID and iscsiadm utilities from
inside a container.

## Example Usage

For any container orchestrator on a host that wishes to run containers that
should utilize remote iscsi targets, this binary should be built to target the
host and then run on it as a service.

You can use the ``setup_iscsi.sh`` file to download and install a pre-build
binary of iscsi-recv for linux hosts.

```bash
$ ./setup_iscsi.sh
$ ls /var/datera/
csi-iscsi.sock  iscsi-recv
```

The sockfile created by iscsi-recv should be mounted into any container
needing to perform iscsi target logins.  The preferred mount location is
``/iscsi-socket/iscsi.sock``, which is the default location that iscsi-send
will look when connecting to the socket.

Inside the container should be a copy of iscsi-send (built for the container's
image) as well as the iscsiadm wrapper script located in the /assets folder
of this repository.  The iscsiadm wrapper script simply redirects calls to
iscsiadm to instead call iscsi-send with the appropriate arguments.

## Debugging

The most common issues with communication are the following

* iscsi-recv is not running on the host
* The sockfile from the host is not mounted into the container
* The sockfile from the host is not mounted to the expected location within the container
* iscsiadm and iscsid are not installed and running on the host
* iscsiadm and iscsid are running inside the container (they should not be installed at all)
