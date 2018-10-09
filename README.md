# Datera CSI Volume Plugin

This plugin uses Datera storage backend as distributed data storage for containers.

## Kubernetes Installation/Configuration (Kubernetes v1.12+ required)

```bash
$ git clone http://github.com/Datera/kubernetes-driver
```

Modify csi/csi-datera-v0.1.0.yaml and update the values for the following
environment variables in the yaml:

`DAT_MGMT`   -- The management IP of the Datera system
`DAT_USER`   -- The username of your Datera account
`DAT_PASS`   -- The password for your Datera account
`DAT_TENANT` -- The tenant to use with your Datera account
`DAT_API`    -- The API version to use when communicating (should be 2.2,
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
$ kubectl create -f csi/csi-datera-v0.1.0.yaml
```
