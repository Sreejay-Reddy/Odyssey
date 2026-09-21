package socket

import (
	"io"
	"fmt"
	"net"
	"errors"
	"encoding/binary"
	"encoding/json"

	"github.com/sreejay-reddy/odyssey/protocol/gen/go"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"

	"capnproto.org/go/capnp/v3"
)

func DecodeRegistry(msg *capnp.Message, r *registry.Registry) error {
	root, err := protocol.ReadRootRegistryMessage(msg)
	if err != nil {
		return err
	}

	sdkID, err := root.SdkID()
	if err != nil {
		return err
	}

	sessionID, err := root.SessionID()
	if err != nil {
		return err
	}

	if len(sdkID) != 16 {
		return errors.New("invalid SDK ID")
	}

	if len(sessionID) != 16 {
		return errors.New("invalid session ID")
	}

	targets, err := root.Targets()
	if err != nil {
		return err
	}

    for i := 0; i < targets.Len(); i++ {
		target := targets.At(i)

		targetID := target.TargetID()

		targetName, err := target.Name()
		if err != nil {
			return err
		}

		functionName, err := target.FunctionName()
		if err != nil {
			return err
		}

		ttlMS := target.TtlMS()

        err = r.Add(registry.Registered{
            Target:       targetName,
            FunctionName: functionName,
            TargetID:     targetID,
			TTLMS: 		  ttlMS,
        })
        if err != nil {
            return err
        }
    }

    return nil
}

func DecodeResult(conn net.Conn, buf []byte) (Result, error) {
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return Result{}, err
	}

	const headerSize = 1 + 16 + 16 + 8 + 4

	if len(buf) < headerSize {
		return Result{}, fmt.Errorf("result too short")
	}

	offset := 0

	result := Result{}

	result.Version = buf[offset]
	offset++

	copy(result.SDKID[:], buf[offset:offset+16])
	offset += 16

	copy(result.SessionID[:], buf[offset:offset+16])
	offset += 16

	result.BatchID = binary.BigEndian.Uint64(buf[offset:])
	offset += 8

	executionCount := binary.BigEndian.Uint32(buf[offset:])
	offset += 4

	result.Executions = make([]ResultExecution, 0, executionCount)

	for i := uint32(0); i < executionCount; i++ {
		if offset+2 > len(buf) {
			return Result{}, fmt.Errorf("truncated result key length")
		}

		keyLength := int(binary.BigEndian.Uint16(buf[offset:]))
		offset += 2

		if offset+keyLength > len(buf) {
			return Result{}, fmt.Errorf("truncated result key")
		}

		key := string(buf[offset : offset+keyLength])
		offset += keyLength

		if offset+4 > len(buf) {
			return Result{}, fmt.Errorf("truncated result target ID")
		}

		targetID := binary.BigEndian.Uint32(buf[offset:])
		offset += 4

		if offset+1 > len(buf) {
			return Result{}, fmt.Errorf("truncated result status")
		}

		status := ExecutionStatus(buf[offset])
		offset++

		if offset+4 > len(buf) {
			return Result{}, fmt.Errorf("truncated result payload length")
		}

		resultLength := int(binary.BigEndian.Uint32(buf[offset:]))
		offset += 4

		if offset+resultLength > len(buf) {
			return Result{}, fmt.Errorf("truncated result payload")
		}

		executionResult := json.RawMessage(buf[offset : offset+resultLength])
		offset += resultLength

		result.Executions = append(result.Executions, ResultExecution{
			Key:      key,
			TargetID: targetID,
			ExecutionResult: executionResult,
			Status:   status,
		})
	}

	if offset != len(buf) {
		return Result{}, fmt.Errorf(
			"unexpected trailing data: %d bytes",
			len(buf)-offset,
		)
	}

	return result, nil
}