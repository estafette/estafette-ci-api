from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LogObject(_message.Message):
    __slots__ = ["bucket", "path"]
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    bucket: str
    path: str
    def __init__(self, bucket: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class Migration(_message.Message):
    __slots__ = ["error", "from_log_object", "id", "to_log_object"]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    FROM_LOG_OBJECT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    TO_LOG_OBJECT_FIELD_NUMBER: _ClassVar[int]
    error: MigrationError
    from_log_object: LogObject
    id: int
    to_log_object: LogObject
    def __init__(self, id: _Optional[int] = ..., from_log_object: _Optional[_Union[LogObject, _Mapping]] = ..., to_log_object: _Optional[_Union[LogObject, _Mapping]] = ..., error: _Optional[_Union[MigrationError, _Mapping]] = ...) -> None: ...

class MigrationError(_message.Message):
    __slots__ = ["code", "errors", "message"]
    CODE_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: int
    errors: _containers.RepeatedCompositeFieldContainer[MigrationErrorDetail]
    message: str
    def __init__(self, code: _Optional[int] = ..., message: _Optional[str] = ..., errors: _Optional[_Iterable[_Union[MigrationErrorDetail, _Mapping]]] = ...) -> None: ...

class MigrationErrorDetail(_message.Message):
    __slots__ = ["domain", "message", "reason"]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    domain: str
    message: str
    reason: str
    def __init__(self, domain: _Optional[str] = ..., reason: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class Request(_message.Message):
    __slots__ = ["from_log_object", "id", "to_log_object"]
    FROM_LOG_OBJECT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    TO_LOG_OBJECT_FIELD_NUMBER: _ClassVar[int]
    from_log_object: LogObject
    id: int
    to_log_object: LogObject
    def __init__(self, id: _Optional[int] = ..., from_log_object: _Optional[_Union[LogObject, _Mapping]] = ..., to_log_object: _Optional[_Union[LogObject, _Mapping]] = ...) -> None: ...

class Response(_message.Message):
    __slots__ = ["batches", "migrations"]
    BATCHES_FIELD_NUMBER: _ClassVar[int]
    MIGRATIONS_FIELD_NUMBER: _ClassVar[int]
    batches: int
    migrations: _containers.RepeatedCompositeFieldContainer[Migration]
    def __init__(self, migrations: _Optional[_Iterable[_Union[Migration, _Mapping]]] = ..., batches: _Optional[int] = ...) -> None: ...
