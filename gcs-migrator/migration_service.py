import logging
import math

from google.cloud import storage
from typing import List

import gcs
import migrate_pb2
import migrate_pb2_grpc


def object_path(self):
    return f'gs://{self.bucket}/{self.path}'


migrate_pb2.LogObject.uri = object_path


class Servicer(migrate_pb2_grpc.ServiceServicer):

    def __init__(self, logger: logging.Logger):
        self.max_batch_size = 1000
        self.storage_client = storage.Client()
        self.logger = logger

    def Migrate(self, request_iterator, context):
        metadict = dict(context.invocation_metadata())
        message_unique_id = metadict['messageuniqueid']
        self.logger.info('Received Migrate request', extra={'messageuniqueid': message_unique_id})
        migrations = []
        m_batch = []
        batch_size = 1
        total = 0
        for req in request_iterator:
            if batch_size > self.max_batch_size:
                migrations = migrations + self.migrate(m_batch, message_unique_id=message_unique_id)
                m_batch = []
                batch_size = 1
            m_batch.append(req)
            batch_size += 1
            total += 1
        if len(m_batch) > 0:
            migrations = migrations + self.migrate(m_batch, message_unique_id=message_unique_id)
        response = migrate_pb2.Response(migrations=migrations, batches=math.ceil(total / self.max_batch_size))
        return response

    def migrate(self, m_batch: List[migrate_pb2.Request], message_unique_id: str = None):
        operation = []
        batcher = gcs.Batcher(client=self.storage_client)
        with batcher:
            for migrate_request in m_batch:
                source_bucket = self.storage_client.bucket(migrate_request.from_log_object.bucket)
                destination_bucket = self.storage_client.bucket(migrate_request.to_log_object.bucket)
                source_blob = source_bucket.blob(migrate_request.from_log_object.path)
                response = source_bucket.copy_blob(
                    source_blob, destination_bucket, migrate_request.to_log_object.path
                )
                operation.append((migrate_request, response))
        migrations = []
        for req, resp in operation:
            migrations.append(migrate_pb2.Migration(
                id=req.id,
                from_log_object=req.from_log_object,
                to_log_object=req.to_log_object,
                error=resp.error,
            ))
            if resp.error is not None:
                self.logger.error('Error migrating log object',
                                  extra={
                                      'messageuniqueid': message_unique_id,
                                      'from_log_object': req.from_log_object.uri(),
                                      'to_log_object': req.to_log_object.uri(),
                                      'error': resp.error
                                  })
        return migrations
