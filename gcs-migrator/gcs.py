import json

from google.cloud.storage import batch
from google.protobuf.json_format import ParseDict

import migrate_pb2


class Batcher(batch.Batch):
    def _finish_futures(self, responses):
        if len(self._target_objects) != len(responses):
            raise ValueError('Expected a response for every request.')
        for target_object, sub_response in zip(self._target_objects, responses):
            if not 200 <= sub_response.status_code < 300 and target_object is not None:
                target_object._properties = {}
                target_object.error = ParseDict(json.loads(sub_response.content.decode('utf-8'))['error'],
                                                migrate_pb2.GcsError())
            elif target_object is not None:
                try:
                    target_object._properties = sub_response.json()
                except ValueError:
                    target_object._properties = sub_response.content
                target_object.error = None
