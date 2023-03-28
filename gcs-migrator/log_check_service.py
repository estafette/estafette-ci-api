import logging
import itertools

from google.cloud import storage
from typing import List
from joblib import Parallel, delayed

import gcs
import migrate_pb2
import migrate_pb2_grpc


class Servicer(migrate_pb2_grpc.LogCheckServicer):

    def __init__(self, logger: logging.Logger):
        self.max_batch_size = 1000
        self.max_parallel_requests = 15
        self.storage_client = storage.Client()
        self.logger = logger

    def Check(self, request_iterator, context):
        try:
            self.logger.info('checking log objects')
            batches: List[List[migrate_pb2.LogObject]] = [[]]
            total = 0
            for req in request_iterator:
                batch_id = total // self.max_batch_size
                if batch_id >= len(batches):
                    batches.append([])
                batches[batch_id].append(req)
                total += 1
            checks = Parallel(n_jobs=self.max_parallel_requests, prefer='threads')(
                delayed(self.check)(m_batch=batch) for _, batch in
                enumerate(batches))
            response = migrate_pb2.LogCheckResponse(log_objects=list(itertools.chain(*checks)), batches=len(batches))
            self.logger.info('checked log objects', extra={'logObjects': total, 'batches': len(batches)})
            return response
        except Exception as e:
            self.logger.error('Error checking log objects', extra={'error': str(e)})
            raise e

    def check(self, m_batch: List[migrate_pb2.LogObject]):
        self.logger.debug('migrating batch', extra={'batchSize': len(m_batch)})
        batcher = gcs.Batcher(client=self.storage_client)
        checks = []
        check_results = []
        with batcher:
            for log_check_request in m_batch:
                source_bucket = self.storage_client.bucket(log_check_request.bucket)
                source_blob = source_bucket.blob(log_check_request.path)
                response = source_blob.exists()
                checks.append((log_check_request, response))
        for req, resp in checks:
            check_results.append(migrate_pb2.LogObjectStatus(
                log_object=req,
                exists=resp
            ))
        return check_results
