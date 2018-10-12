#!/usr/bin/env python3

import contextlib
import io
import os
import shlex
import subprocess
import tempfile
import random
import string
import time
# import sys

import ruamel.yaml
from dfs_sdk import scaffold

YAML = ruamel.yaml.YAML(typ='safe', pure=True)
YAML.default_flow_style = False
YAMLS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "yamls")
VERBOSE = False

VERSIONS = ["v0.1.0", "latest"]

CHARS = string.ascii_letters

SC_TMPL = """
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: no-name
  # annotations:
  #   storageclass.kubernetes.io/is-default-class: "true"
provisioner: com.daterainc.csi.dsp
"""


def dprint(*args, **kwargs):
    if VERBOSE:
        print(*args, **kwargs)


def exec_command(cmd):
    dprint("Running: {}".format(cmd))
    return subprocess.check_output(shlex.split(cmd))


def get_yaml(yml):
    return os.path.join(YAMLS, yml)


def read_yaml(yml):
    """Returns a list of dicts each being a yaml document in the file"""
    with io.open(get_yaml(yml), 'r') as f:
        return list(YAML.load_all(f))


def gen_sc_name():
    return "-".join(
        ("dat", "sc", "".join((random.choice(CHARS) for _ in range(5)))))


def mk_sc(**params):
    yml = YAML.load(SC_TMPL)
    name = gen_sc_name()
    yml["metadata"]["name"] = name
    yml["parameters"] = params
    return name, yml


def kexe(cmd):
    cmd = " ".join(("kubectl", cmd))
    return exec_command(cmd)


def kopf(cmd, yml):
    name = None
    with tempfile.NamedTemporaryFile(delete=False) as f:
        YAML.dump_all(yml, f)
        name = f.name
    cmd = cmd.format(name)
    try:
        result = exec_command(cmd)
    finally:
        os.remove(name)
    return result


def kcreate(yml):
    """yml is the yaml object, NOT the filename"""
    cmd = "kubectl create -f {}"
    result = kopf(cmd, yml)
    dprint(result)
    return result


def kdelete(yml):
    """yml is the yaml object, NOT the filename"""
    cmd = "kubectl delete -f"
    result = kopf(cmd, yml)
    dprint(result)
    return result


@contextlib.contextmanager
def kresources(yml):
    delete = []
    for resource in yml:
        dprint("Creating k8s resource: {}".format(resource['kind']))
        kcreate(resource)
        delete.append(resource)
    try:
        yield
    finally:
        for resource in delete:
            kdelete(resource)


def test_install():
    for v in VERSIONS:
        yml = read_yaml("csi-datera-{}.yaml".format(v))
        with kresources(yml):
            timeout = 10
            while True:
                if not timeout:
                    print(kexe("get pods --namespace kube-system"))
                    raise ValueError(
                        "Timeout reached and some services did not reach "
                        "running state")
                out = kexe(
                    "get pods --namespace kube-system | awk '{print $3}'")
                for line in out.splitlines()[1:]:
                    if line.strip() != "Running":
                        timeout -= 1
                        time.sleep(1)
                        continue


def main(args):
    test_install()


if __name__ == '__main__':
    parser = scaffold.get_argparser()
    args = parser.parse_args()
    main(args)
