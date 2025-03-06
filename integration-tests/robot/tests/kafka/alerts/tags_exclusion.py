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

def check_parameters_are_specified(environ, *variable_names) -> bool:
    for variable in variable_names:
        if not environ.get(variable):
            return False
    return True


def check_parameters_are_present(environ, *variable_names) -> bool:
    for variable in variable_names:
        if environ.get(variable) is None:
            return False
    return True


def get_excluded_tags(environ) -> list:
    if not check_parameters_are_specified(environ,
                                          'PROMETHEUS_URL') \
            or check_parameters_are_present(environ, 'EXTERNAL_KAFKA'):
        return ['prometheus']
