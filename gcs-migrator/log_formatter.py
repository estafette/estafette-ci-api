import datetime
import itertools
import logging
import socket

from google.protobuf.json_format import MessageToJson
from pythonjsonlogger import jsonlogger


class V3(jsonlogger.JsonFormatter):
    sequence_id = itertools.count()

    def add_fields(self, log_record, record, message_dict):
        super(V3, self).add_fields(log_record, record, message_dict)
        log_record['logformat'] = 'v3'
        log_record['environment'] = 'production'
        log_record['platform'] = 'k8s'
        log_record['sequenceid'] = next(V3.sequence_id)
        # log_record['messageuniqueid'] =
        log_record['messagetype'] = 'estafette-gcs-migrator'
        log_record['messagetypeversion'] = '0.0.0'
        log_record['source'] = {
            'appgroup': 'estafette-ci',
            'appname': 'gcs-migrator',
            'appversion': '0.1.0',
            'hostnam': socket.gethostname()
        }
        now = datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%S.%fZ')
        log_record['timestamp'] = now
        if log_record.get('level'):
            log_record['loglevel'] = log_record['level'].upper()
        else:
            log_record['loglevel'] = record.levelname


class Console(logging.Formatter):

    def format(self, record: logging.LogRecord) -> str:
        default_attrs = logging.LogRecord('', 0, '', 0, None, None, None).__dict__.keys()
        extras = sorted(set(record.__dict__.keys()) - default_attrs)
        log_items = ['%(name)s %(asctime)s %(levelname)s: %(message)s']
        for attr in extras:
            if attr == 'error':
                log_items.append(f'{attr}="\n%({attr})s\n"')
                record.error = MessageToJson(record.error)
            else:
                log_items.append(f'{attr}="%({attr})s"')
        format_str = f'{" ".join(log_items)}'
        self._style._fmt = format_str

        return super().format(record)
