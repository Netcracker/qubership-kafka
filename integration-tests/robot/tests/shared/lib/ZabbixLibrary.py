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
import datetime
import xml.etree.ElementTree
from robot.api import logger
from pyzabbix import ZabbixAPI
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


class ZabbixLibrary:
    def __init__(self, url=None, user=None, password=None):
        if url:
            self.zapi = ZabbixAPI(url)
            if user and password:
                self.zapi.session.verify = False
                self.zapi.login(user, password)
        self._start_timestamp: int = int(datetime.datetime.now().timestamp())

    def get_host_id_by_name(self, host_name):
        result = self.zapi.do_request('host.get', {'filter': {"host": [host_name]}})['result']
        return result[0]['hostid'] if result else None

    def get_relevant_event_names_by_host_id(self, host_id: str, clock_filter_enabled=True) -> list:
        events = self.zapi.do_request('event.get', {'hostids': [host_id], 'sortfield': 'clock'})['result']
        filtered_event_names = []
        if clock_filter_enabled:
            for event in events:
                if int(event['clock']) > self._start_timestamp:
                    filtered_event_names.append(event['name'])
        else:
            filtered_event_names = [event['name'] for event in events]
        return filtered_event_names

    def check_that_problem_name_is_presented_for_host_name(self, host_name: str, problem_msg: str,
                                                           clock_filter_enabled=True) -> bool:
        host_id = self.get_host_id_by_name(host_name)
        if not host_id:
            return False
        event_names = self.get_relevant_event_names_by_host_id(host_id, clock_filter_enabled=clock_filter_enabled)
        return True if problem_msg in event_names else False

    def is_zabbix_problem_exist(self, host_name, problem_name):
        """
        Returns True if there is zabbix problem with given name and which is linked
        to given zabbix host.
        :param host_name: zabbix host name
        :param problem_name: zabbix problem name
        :return: bool
        """
        host_id = self.get_host_id_by_name(host_name)
        if not host_id:
            return False
        problems = self.zapi.do_request('problem.get', {'hostids': host_id})['result']
        if not problems:
            return False
        for problem in problems:
            if problem['name'] == problem_name:
                return True
        return False

    def create_host_group(self, group_name: str):
        self.zapi.do_request('hostgroup.create', {'name': group_name})

    def delete_host_group(self, group_name: str):
        host_group_id = self.get_host_group_id(group_name)
        if host_group_id:
            self.zapi.do_request('hostgroup.delete', [host_group_id])

    def get_host_group_id(self, host_group_name: str):
        result = self.zapi.do_request('hostgroup.get', {'filter': {'name': [host_group_name]}})['result']
        if not result:
            logger.debug('There is no group with such name')
            return None
        return result[0]['groupid']

    def create_host(self, host_name: str, host_group_name: str, template_name: str):
        host_group_id = self.get_host_group_id(host_group_name)
        template_id = self.get_template_id(template_name)
        self.zapi.do_request(
            'host.create', {'host': host_name,
                            'groups': [{'groupid': host_group_id}],
                            'templates': [{'templateid': template_id}],
                            'interfaces': [{'ip': '', 'dns': 'zabbix-agent', 'port': '10050', 'type': 1, 'main': 1,
                                            'useip': 0}]})  # we use DNS name

    def get_host_id(self, host_name: str):
        result = self.zapi.do_request('host.get', {'filter': {'host': [host_name]}})['result']
        if not result:
            logger.debug('There is no host with such name')
            return None
        return result[0]['hostid']

    def delete_host_by_name(self, host_name: str):
        host_id = self.get_host_id(host_name)
        if host_id:
            self.zapi.do_request('host.delete', [host_id])

    def add_macros_to_host(self, host_name: str, **macroses) -> None:
        """
        Adds macros to given zabbix host. All named variables are
        used as key-value macros. For example, in Robot Framework:
        Add Macros To Host | Elasticsearch host | CPU_LIMIT=95
        Macros CPU_LIMIT with value - "95" will be added to
        "Elasticsearch host" zabbix host.
        :param host_name: zabbix host name
        :param macroses: zabbix host macros dictionary
        :return: void
        """
        host_id = self.get_host_id_by_name(host_name)
        macros = []
        for key, value in macroses.items():
            macros.append({'macro': self._wrap_macros_key(key), 'value': value})
        self.zapi.do_request('host.massadd', {'hosts': [{'hostid': host_id}], 'macros': macros})

    @staticmethod
    def _wrap_macros_key(key: str):
        return f'{{${key}}}'

    def _load_template_internal(self, source_xml):
        self.zapi.do_request('configuration.import', {'format': 'xml',
                                                      'rules': {'items': {'createMissing': True},
                                                                'templates': {'createMissing': True},
                                                                'groups': {'createMissing': True},
                                                                'applications': {'createMissing': True},
                                                                'triggers': {'createMissing': True}},
                                                      'source': source_xml})

    def get_template_id(self, template_name: str):
        result = self.zapi.do_request('template.get', {'filter': {'host': [template_name]}})['result']
        if not result:
            logger.debug('There is no template with such name')
            return None
        return result[0]['templateid']

    def delete_template_by_name(self, template_name: str):
        template_id = self.get_template_id(template_name)
        if template_id:
            self.zapi.do_request('template.delete', [template_id])

    def load_template(self, host_group, template_name, template_visible_name, application_name, file_path):
        """
        Method uses given params to create zabbix import data which contains host group, template,
        application, linked items and triggers. And then it import this data to zabbix.
        :param host_group: zabbix host group
        :param template_name: zabbix template name
        :param template_visible_name: zabbix template visible name
        :param application_name: zabbix application name
        :param file_path: path to zabbix template xml file
        :return: void
        """
        source_xml = generate_configuration_template(host_group,
                                                     template_name,
                                                     template_visible_name,
                                                     application_name,
                                                     file_path)
        self._load_template_internal(source_xml)

    @staticmethod
    def check_that_parameters_are_presented(*variable_names) -> bool:
        for variable in variable_names:
            if not os.getenv(variable):
                return False
        return True


def generate_configuration_template(host_group: str,
                                    template_name: str,
                                    template_visible_name: str,
                                    application_name: str,
                                    file_path: str):
    """
    Method uses ZabbixTemplateBuilder class and given parameters to change
    zabbix import data xml and returns result string.
    :param host_group: zabbix host group
    :param template_name: zabbix template name
    :param template_visible_name: zabbix template visible name
    :param application_name: zabbix application name
    :param file_path: path to zabbix template xml file
    :return: string
    """
    ztp = ZabbixTemplateBuilder(file_path)
    return (ztp
            .set_group_name(host_group)
            .set_template_name(template_name)    # should be the same with host name for triggers
            .set_template_visible_name(template_visible_name)
            .set_application_name(application_name)
            .build())


class ZabbixTemplateBuilder:
    def __init__(self, path):
        self.path = path
        self.et = xml.etree.ElementTree.parse(path)

        self.group_name = (
            self.et.getroot()
                .findall('groups')[0]
                .findall('group')[0]
                .findall('name')[0])

        self._template_root = (
            self.et.getroot()
                .findall('templates')[0]
                .findall('template')[0])

        self.template_template = self._template_root.findall('template')[0]

        self.template_group_name = (
            self._template_root
                .findall('groups')[0]
                .findall('group')[0]
                .findall('name')[0])

        self.template_name = self._template_root.findall('name')[0]
        self.application_name = (
            self._template_root
                .findall('applications')[0]
                .findall('application')[0]
                .findall('name')[0])

        self.expressions = [trigger.findall('expression')[0] for trigger in self.et.getroot().findall('triggers')[0]]
        self.recovery_expressions = [trigger.findall('recovery_expression')[0]
                                     for trigger in self.et.getroot().findall('triggers')[0]]

        self.item_application_names = [item.findall('applications')[0].findall('application')[0].findall('name')[0]
                                       for item in self._template_root.findall('items')[0]]

        triggers = self.et.getroot().findall('triggers')[0]
        triggers_dependencies = list(filter(None, [self.get_attributes_chain(trigger, ['dependencies', 'dependency'])
                                                   for trigger in triggers]))

        self.dependency_expressions = [dependency.findall('expression')[0] for dependency in triggers_dependencies]
        self.dependency_recovery_expressions = [dependency.findall('recovery_expression')[0]
                                                for dependency in triggers_dependencies]

    @staticmethod
    def get_attributes_chain(element, attrs: list):
        res = element
        for attr in attrs:
            res = res.findall(attr)
            if not res:
                return None
            res = res[0]
        return res

    def set_group_name(self, group_name):
        self.group_name.text = group_name
        self.template_group_name.text = group_name
        return self

    def set_template_name(self, template_name):
        self.template_template.text = template_name
        self.set_host_name_for_triggers(template_name)
        return self

    def set_template_visible_name(self, template_visible_name):
        self.template_name.text = template_visible_name
        return self

    def set_host_name_for_triggers(self, host_name):
        wrapped_host_name = f'{{{host_name}:'
        old_host_name = self._get_old_host_name()
        wrapped_old_host_name = f'{{{old_host_name}:'
        for expression in self.expressions:
            new_text = expression.text.replace(wrapped_old_host_name, wrapped_host_name)
            expression.text = new_text
        for recovery_expression in self.recovery_expressions:
            if recovery_expression.text:
                new_text = recovery_expression.text.replace(wrapped_old_host_name, wrapped_host_name)
                recovery_expression.text = new_text
        for dependency_expression in self.dependency_expressions:
            if dependency_expression.text:
                new_text = dependency_expression.text.replace(wrapped_old_host_name, wrapped_host_name)
                dependency_expression.text = new_text
        for dependency_recovery_expression in self.dependency_recovery_expressions:
            if dependency_recovery_expression.text:
                new_text = dependency_recovery_expression.text.replace(wrapped_old_host_name, wrapped_host_name)
                dependency_recovery_expression.text = new_text
        return self

    def _get_old_host_name(self):
        text = self.expressions[0].text
        res = text[1:text.find(':')]
        return res

    def set_application_name(self, application_name):
        self.application_name.text = application_name
        for item_application_name in self.item_application_names:
            item_application_name.text = application_name
        return self

    def build(self):
        return xml.etree.ElementTree.tostring(self.et.getroot(), encoding='UTF-8').decode('UTF-8')