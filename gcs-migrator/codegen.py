#!/usr/bin/env python3
from grpc_tools import protoc

protoc.main((
    '',
    '--proto_path=gcs-migrator/protos',
    '--python_out=gcs-migrator',
    '--pyi_out=gcs-migrator',
    '--grpc_python_out=gcs-migrator',
    'gcs-migrator/protos/migrate.proto',
))
