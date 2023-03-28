import os
import grpc
import logging
import signal

from concurrent import futures

import migrate_pb2
import migrate_pb2_grpc
import log_formatter
import migration_service
import log_check_service
import health_service

logger = logging.getLogger('gcs-migrator')
logger.setLevel(logging.INFO)
logHandler = logging.StreamHandler()
if os.getenv('ESTAFETTE_LOG_FORMAT') == 'v3':
    formatter = log_formatter.V3()
    logger.addHandler(logHandler)
else:
    formatter = log_formatter.Console()
logHandler.setFormatter(formatter)
logger.addHandler(logHandler)


def object_path(self):
    return f'gs://{self.bucket}/{self.path}'


migrate_pb2.LogObject.uri = object_path


def serve():
    addr = '[::]:50051'
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    migrate_pb2_grpc.add_ServiceServicer_to_server(migration_service.Servicer(logger), server)
    migrate_pb2_grpc.add_HealthServicer_to_server(health_service.Servicer(logger), server)
    migrate_pb2_grpc.add_LogCheckServicer_to_server(log_check_service.Servicer(logger), server)
    server.add_insecure_port(addr)
    server.start()
    logger.info('Server started, listening on {0}'.format(addr))
    server.wait_for_termination()


def graceful_handler(signum, _):
    logger.info('Shutting down, received signal %d) %s', signum, signal.Signals(signum).name)
    exit(0)


signal.signal(signal.SIGINT, graceful_handler)
signal.signal(signal.SIGTERM, graceful_handler)

if __name__ == '__main__':
    serve()
