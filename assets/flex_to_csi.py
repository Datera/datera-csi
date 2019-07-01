#!/usr/bin/env python

"""
DO NOT USE

WIP flex to csi migration script

This was intended to be the way customers migrate existing volumes from
a Flex install to a CSI install.

Design:

    1. Utilize the kubernetes python client to list all current persistent
       volume claims for all namespaces
    2. Use a regex to match the naming convention of the PVC backing volume
       which indicates it was originally provisioned with Flex
    3. Rename the backing volume
    4. Delete the PVC (which should no longer be able to delete the volume)
    5. Create a new PVC using the CSI plugin and specify the new volume name.
       The CSI plugin always tries to find a volume with the provided name
       before provisioning a new one.
    6. Now the PVC is ready to be used by an application.
"""

from __future__ import (print_function, division, absolute_import,
                        unicode_literals)

import argparse
import re
import sys

from kubernetes import client, config
from dfs_sdk import scaffold

RE = re.compile(r"k8s_(?P<ns>.+?)_(?P<vid>.+)")


def main(args):

    raise NotImplementedError("This script has not been fully implemented")
    api = scaffold.get_api()
    scaffold.print_config()

    api.app_instances.list()

    config.load_kube_config()

    v1 = client.CoreV1Api()
    print("Finding Flex volumes")
    flex_vols = []
    ret = v1.list_persistent_volume_claim_for_all_namespaces(watch=False)
    for elem in ret.items:
        if RE.match(elem.spec.volume_name):
            flex_vols.append({"name": elem.spec.volume_name,
                              "size": elem.status.capacity.storage})
        print("{}".format(elem))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--dry-run')
    args = parser.parse_args()
    sys.exit(main(args))
