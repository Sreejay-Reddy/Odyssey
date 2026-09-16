import struct

from .protocol import Message, Execution, Result

def decode_message(data):
    offset = 0

    version = data[offset]
    offset += 1

    message_type = data[offset]
    offset += 1

    flags = struct.unpack_from(">H", data, offset)[0]
    offset += 2

    batch_id = struct.unpack_from(">Q", data, offset)[0]
    offset += 8

    execution_count = struct.unpack_from(">I", data, offset)[0]
    offset += 4

    executions = []

    for _ in range(execution_count):
        key_length = struct.unpack_from(">H", data, offset)[0]
        offset += 2

        key = data[offset:offset + key_length].decode("utf-8")
        offset += key_length

        target_id = struct.unpack_from(">I", data, offset)[0]
        offset += 4

        input_length = struct.unpack_from(">I", data, offset)[0]
        offset += 4

        input_data = data[offset:offset + input_length]
        offset += input_length

        executions.append(
            Execution(
                key=key,
                target_id=target_id,
                input=input_data,
            )
        )

    return Message(
        version=version,
        type=message_type,
        flags=flags,
        batch_id=batch_id,
        executions=executions,
    )

def encode_result(result: Result) -> bytes:
    data = bytearray()

    data += struct.pack(">B", result.version)

    if len(result.sdk_id) != 16:
        raise ValueError("sdk_id must be exactly 16 bytes")

    if len(result.session_id) != 16:
        raise ValueError("session_id must be exactly 16 bytes")

    data += result.sdk_id
    data += result.session_id

    data += struct.pack(">Q", result.batch_id)
    data += struct.pack(">I", len(result.executions))

    for execution in result.executions:
        key = execution.key.encode("utf-8")

        data += struct.pack(">H", len(key))
        data += key

        data += struct.pack(">I", execution.target_id)

        data += struct.pack(">B", execution.status)

        data += struct.pack(">I", len(execution.execution_result))
        data += execution.execution_result

    return bytes(data)