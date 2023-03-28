import migrate_pb2
import migrate_pb2_grpc


class Servicer(migrate_pb2_grpc.HealthServicer):

    def __init__(self, logger):
        self.logger = logger

    def Check(self, request, context):
        response = migrate_pb2.HealthCheckResponse(status=migrate_pb2.HealthCheckResponse.SERVING)
        return response
