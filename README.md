# Datera CSI Volume Plugin

This plugin uses Datera storage backend as distributed data storage for containers.

## Kubernetes Installation/Configuration (Kubernetes v1.13+ required)

### NOTE: Container-based ISCSID is no longer supported

You can use the iscsid on the host following steps (These MUST be performed
before installing the CSI plugin):

First install iscsid on the kubernetes hosts

Ubuntu
```bash
$ apt install open-iscsi
```

Centos
```bash
$ yum install iscsi-initiator-utils
```

Verify iscsid is running:
```bash
$ ps -ef | grep iscsid
root     12494   996  0 09:41 pts/2    00:00:00 grep --color=auto iscsid
root     13326     1  0 Dec17 ?        00:00:01 /sbin/iscsid
root     13327     1  0 Dec17 ?        00:00:05 /sbin/iscsid
```

Clone the datera-csi repository
```bash
$ git clone http://github.com/Datera/datera-csi
```

Run the iscsi-recv service installer
```bash
$ ./assets/setup_iscsi.sh
[INFO] Dependency checking
[INFO] Downloading iscsi-recv
[INFO] Verifying checksum
[INFO] Changing file permissions
[INFO] Registering iscsi-recv service
Created symlink from /etc/systemd/system/multi-user.target.wants/iscsi-recv.service to /lib/systemd/system/iscsi-recv.service.
[INFO] Starting iscsi-recv service
[INFO] Verifying service started correctly
root      4879     1  0 19:50 ?        00:00:00 /tmp/iscsi-recv -addr unix:////tmp/csi-iscsi.sock
```

Check that the iscsi-recv service is running
```bash
$ systemctl --all | grep iscsi-recv
iscsi-recv.service       loaded    active     running   iscsi-recv container to host iscsiadm adapter service
```

```bash
$ git clone http://github.com/Datera/datera-csi
```

Modify deploy/kubernetes/with-host-iscsid/csi-datera-latest.yaml and update the
values for the following environment variables in the yaml:

* `DAT_MGMT`   -- The management IP of the Datera system
* `DAT_USER`   -- The username of your Datera account
* `DAT_PASS`   -- The password for your Datera account
* `DAT_TENANT` -- The tenant to use with your Datera account
* `DAT_API`    -- The API version to use when communicating (should be 2.2,
                currently the only version the plugin supports)

There are two locations for each value within the yaml that should be modified

Additionally the yaml comes with a built-in StorageClass (dat-block-storage).
Feel free to modify or remove it depending on deployment needs.

Volume Parameters can be placed within the ``parameters`` section of the
StorageClass

In the following example we configure volumes with a replica of 3 and a QoS of
1000 IOPS max.  All parameters must be strings (pure numbers and booleans
should be enclosed in quotes)

```yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: dat-block-storage
  namespace: kube-system
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: io.datera.csi.dsp
parameters:
  replica_count: "3"
  total_iops_max "1000"
```

Here are a list of supported parameters for the plugin:

Name                   |     Default
----------------       |     ------------
``replica_count``      |     ``3``
``placement_mode``     |     ``hybrid``
``ip_pool``            |     ``default``
``template``           |     ``""``
``round_robin``        |     ``false``
``read_iops_max``      |     ``0``
``write_iops_max``     |     ``0``
``total_iops_max``     |     ``0``
``read_bandwidth_max`` |     ``0``
``write_bandwidth_max``|     ``0``
``total_bandwidth_max``|     ``0``
``iops_per_gb``        |     ``0``
``bandwidth_per_gb``   |     ``0``
``fs_type``            |     ``ext4``
``fs_args``            |     ``-E lazy_itable_init=0,lazy_journal_init=0,nodiscard -F``
``delete_on_unmount``  |     ``false``


```bash
$ kubectl create -f csi/csi-datera-latest.yaml
```

## Create A Volume

Example PVC

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mypvc
  namespace: default
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: dat-block-storage
```

```bash
$ kubectl create -f pvc.yaml
```

## Create An Application Using the Volume

Save the following as app.yaml
```yaml
kind: Pod
apiVersion: v1
metadata:
  name: my-csi-app
spec:
  containers:
    - name: my-app-image
      image: alpine
      volumeMounts:
      - mountPath: "/data"
        name: my-app-volume
      command: [ "sleep", "1000000" ]
  volumes:
    - name: my-app-volume
      persistentVolumeClaim:
        claimName: mypvc
```

```bash
$ kubectl create -f app.yaml
```

## Optional Secrets

Instead of putting the username/password in the yaml file directly instead
you can use the kubernetes secrets capabilities.

NOTE: This must be done before installing the CSI driver.

First create the secrets.  They're base64 encoded strings.  The two required
secrets are "username" and "password".

Modify and save the below yaml as secrets.yaml
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: datera-secret
  namespace: kube-system
type: Opaque
data:
  # base64 encoded username
  # generate this via "$ echo -n 'your-username' | base64"
  username: YWRtaW4=
  # base64 encoded password
  # generate this via "$ echo -n 'your-password' | base64"
  password: cGFzc3dvcmQ=
```
Then create the secrets

```bash
$ kubectl create -f secrets.yaml
```

Now install the CSI driver like above, but using the "secrets" yaml:

```bash
$ kubectl create -f csi-datera-secrets-latest.yaml
```

The only difference between the "secrets" yaml and the regular yaml is the
use of secrets for the "username" and "password" fields.
