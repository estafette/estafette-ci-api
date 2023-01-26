import itertools
import logging
from typing import List

from google.cloud import storage
from joblib import Parallel, delayed

import gcs
import migrate_pb2
import migrate_pb2_grpc


def object_path(self):
    return f'gs://{self.bucket}/{self.path}'


migrate_pb2.LogObject.uri = object_path


class Servicer(migrate_pb2_grpc.ServiceServicer):

    def __init__(self, logger: logging.Logger):
        self.max_batch_size = 1000
        self.max_parallel_requests = 15
        self.storage_client = storage.Client()
        self.logger = logger

    def Migrate(self, request_iterator, context):
        metadict = dict(context.invocation_metadata())
        message_unique_id = metadict['messageuniqueid']
        self.logger.info('received log objects migration request', extra={'messageuniqueid': message_unique_id})
        batches: List[List[migrate_pb2.Request]] = [[]]
        total = 0
        response: migrate_pb2.Response
        for req in request_iterator:
            batch_id = total // self.max_batch_size
            if batch_id >= len(batches):
                batches.append([])
            batches[batch_id].append(req)
            total += 1
        if total > 0:
            migrations = Parallel(n_jobs=self.max_parallel_requests, prefer='threads')(
                delayed(self.migrate)(m_batch=batch, message_unique_id=message_unique_id) for _, batch in
                enumerate(batches))
            response = migrate_pb2.Response(migrations=list(itertools.chain(*migrations)), batches=len(batches))
            self.logger.info('migrated log objects', extra={'messageuniqueid': message_unique_id, 'logObjects': total,
                                                            'batches': len(batches)})
        else:
            self.logger.warning('no log objects to migrate in request', extra={'messageuniqueid': message_unique_id})
            response = migrate_pb2.Response(migrations=[], batches=0)
        return response

    def migrate(self, m_batch: List[migrate_pb2.Request], message_unique_id: str = None):
        self.logger.debug('migrating batch', extra={'messageuniqueid': message_unique_id, 'batchSize': len(m_batch)})
        operation = []
        batcher = gcs.Batcher(client=self.storage_client)
        migrations = []
        with batcher:
            for migrate_request in m_batch:
                source_bucket = self.storage_client.bucket(migrate_request.from_log_object.bucket)
                destination_bucket = self.storage_client.bucket(migrate_request.to_log_object.bucket)
                source_blob = source_bucket.blob(migrate_request.from_log_object.path)
                response = source_bucket.copy_blob(
                    source_blob, destination_bucket, migrate_request.to_log_object.path
                )
                operation.append((migrate_request, response))
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
