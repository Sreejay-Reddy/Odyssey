from dataclasses import dataclass
from enum import IntEnum


PROTOCOL_VERSION = 1


class MessageType(IntEnum):
    SUBMIT = 1
    ACK = 2 
    RESULT = 3

class ExecutionStatus(IntEnum):
    SUCCESS = 1
    FAILED = 2


@dataclass(slots=True)
class Execution:
    key: str
    target_id: int
    input: bytes


@dataclass(slots=True)
class Message:
    version: int
    type: MessageType
    flags: int
    batch_id: int
    executions: list[Execution]

@dataclass(slots=True)
class ResultExecution:
    key: str
    target_id: int
    execution_result: bytes
    status: ExecutionStatus


@dataclass(slots=True)
class Result:
    version: int
    sdk_id: bytes
    session_id: bytes
    batch_id: int
    executions: list[ResultExecution]