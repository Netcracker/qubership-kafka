#!/usr/bin/python

# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
from time import sleep

import yaml
from kubernetes import client
from kubernetes import config as k8s_config, dynamic
from kubernetes.dynamic.exceptions import NotFoundError

"""
########################################################################################################################
########################################### KUBERNETES CLIENT ##########################################################
########################################################################################################################
"""

try:
    k8s_config.load_incluster_config()
    k8s_client = client.ApiClient()
except k8s_config.ConfigException as e:
    k8s_client = k8s_config.new_client_from_config()

if k8s_client is None:
    import sys

    print("Can't load any kubernetes config.")
    sys.exit(1)

# Creating a dynamic client
client = dynamic.DynamicClient(k8s_client)

# fetching the custom resource definition (CRD) api
crd_api = client.resources.get(
    api_version="apiextensions.k8s.io/v1", kind="CustomResourceDefinition"
)

api_group = os.getenv("API_GROUP", "qubership.org")

"""
########################################################################################################################
###################################### CUSTOM RESOURCE DEFINITION CONTAINER ############################################
########################################################################################################################
"""


class CRD:
    def __init__(self, path_to_crd_file):
        self.crd_path = path_to_crd_file
        with open(path_to_crd_file) as crd_descriptor:
            self.crd_dict = yaml.safe_load(crd_descriptor)
        if not self._validate():
            print("Can't process the CRD, please check logs above")
            exit(1)
        self._extract_metadata()
        self._extract_spec()

    def _validate(self) -> bool:
        if 'kind' not in self.crd_dict:
            print(f'{self.crd_path} doesn\'t have the \'kind\' key, it is not a Kubernetes resource')
            return False
        if 'CustomResourceDefinition' != self.crd_dict['kind']:
            print(f'{self.crd_path} is not a CustomResourceDefinition')
            return False
        return True

    def _extract_metadata(self):
        self.name = self.crd_dict['metadata']['name']
        self.version = self.crd_dict['metadata']['annotations'][f'crd.{api_group}/version']

    def _extract_spec(self):
        self.group = self.crd_dict['spec']['group']
        self.plural = self.crd_dict['spec']['names']['plural']
        self.kind = self.crd_dict['spec']['names']['kind']

    def get_dict(self):
        return self.crd_dict

    def get_name(self):
        return self.name

    def get_version(self):
        return self.version


"""
########################################################################################################################
############################################ VERSIONS COMPARING ########################################################
########################################################################################################################
"""


def _compare_version_part(left_part, right_part) -> int:
    int_left_major = int(left_part)
    int_right_major = int(right_part)
    if int_left_major < int_right_major:
        return 1
    if int_left_major > int_right_major:
        return -1
    return 0


def compare_crd_versions(left_version, right_version) -> int:
    left_major, left_minor, left_patch = left_version.split(".")
    right_major, right_minor, right_patch = right_version.split(".")

    compare_result = _compare_version_part(left_major, right_major)
    if compare_result != 0:
        return compare_result

    compare_result = _compare_version_part(left_minor, right_minor)
    if compare_result != 0:
        return compare_result

    return _compare_version_part(left_patch, right_patch)


"""
########################################################################################################################
############################################ MAIN FUNCTION #############################################################
########################################################################################################################
"""


def str2bool(value: str, _default: bool) -> bool:
    if value == "true":
        return True
    elif value == "false":
        return False
    else:
        print(f"Can't convert {value} to boolean, so return default - {_default}")
        return _default


def _load_crd(crd_name: str):
    try:
        return crd_api.get(name=crd_name)
    except NotFoundError:
        print(f"CRD {crd_name} was not found, so create it")
        return None


def _process_crd(file_path) -> bool:
    """
    The function loads the CRD YAML descriptor from file system and process it according to the cluster state:
        * The CRD might be created, if it is not existed before.
        * The CRD can be replaced, if the local CRD has higher version that CRD on the cluster.
        * The CRD can be skipped, if local & cluster versions are the same.
    :param file_path: path to YAML file with CRD descriptor.
    :return:
        * True if the CRD on the cluster changed (created or replaced)
        * False if the CRD was not changed.
    """
    local_crd = CRD(file_path)
    print(f"Processing local \'{local_crd.get_name()}\' CRD, version={local_crd.get_version()} ...")

    loaded_crd = _load_crd(local_crd.get_name())
    if loaded_crd is None:
        print(f"Creating {local_crd.get_name()} CRD...")
        response = crd_api.create(local_crd.get_dict())
        print(f'Creation response: {response}')
        return True

    print(f"CRD {local_crd.get_name()} was found, so comparing versions...")
    loaded_crd_version = loaded_crd['metadata']['annotations'][f'crd.{api_group}/version']

    compare_result = compare_crd_versions(local_crd.get_version(), loaded_crd_version)
    if compare_result == 0:
        print("Local & Remote CRD versions are same, so no update is needed")
    elif compare_result == 1:
        print(f"Remote CRD has newest version ({loaded_crd_version}), so no update is needed")
    else:
        print(f"Local CRD is newest (version {local_crd.get_version()}), so need to update it")
        # specify `resourceVersion` for local CRD to avoid the following issue:
        #       "metadata.resourceVersion: Invalid value: 0x0: must be specified for an update"
        body = local_crd.get_dict()
        body["metadata"]["resourceVersion"] = loaded_crd['metadata']['resourceVersion']
        crd_api.replace(name=local_crd.get_name(), body=body, strategy="merge")
        return True
    return False

def replace_api_version_in_crd(file_path, old_text, new_text):
    with open(file_path, "r") as file:
        content = file.read()

    new_content = content.replace(old_text, new_text)

    with open(file_path, "w") as file:
        file.write(new_content)

def run():
    crds_to_create = os.getenv("CRDS_TO_CREATE", "")
    if crds_to_create == "":
        print(f"No CRDs passed into CRD init job, skipping CRD upgrade")
        return
    ignore_crd_failure = str2bool(os.getenv("IGNORE_CRD_FAILURE", "true"), True)
    crd_upgrade_waiting_time = int(os.getenv("CRD_UPGRADE_WAITING_TIME", "30"))
    print(f"CRD Init Script Running\nParameters:"
          f"\n\tIGNORE_CRD_FAILURE: {ignore_crd_failure}"
          f"\n\tCRD_UPGRADE_WAITING_TIME: {crd_upgrade_waiting_time}"
          f"\n\tCRDS_TO_CREATE: {crds_to_create}"
          f"\nLocal CRD processing logs are below")
    for root, dirs, files in os.walk("./crds"):
        for file in files:
            path = root + "/" + file
            try:
                if file in crds_to_create:
                    replace_api_version_in_crd(path, "qubership.org", api_group)
                    if _process_crd(path):
                        print(f"Waiting {crd_upgrade_waiting_time} second after CRD upgrade")
                        sleep(crd_upgrade_waiting_time)
            except Exception as any_exception:
                if not ignore_crd_failure:
                    print(f"Critical error while processing {path} file: {any_exception}")
                    exit(1)
                else:
                    print(f"Warring error while processing {path} file: {any_exception}\nContinue processing...")


if __name__ == '__main__':
    run()
