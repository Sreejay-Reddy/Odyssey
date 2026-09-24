package socket

import (
	"errors"

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

func DecodeResult(msg *capnp.Message) (Result, error) {
	root, err := protocol.ReadRootResultMessage(msg)
	if err != nil {
		return Result{}, err
	}

	resultMsg := Result{
		Version: root.ProtocolVersion(),
		BatchID: root.BatchID(),
	}

	results, err := root.Executions()
	if err != nil {
		return Result{}, err
	}

	executions := make([]ResultExecution, 0, results.Len())

	for i:=0; i<results.Len(); i++ {
		result := results.At(i)

		key, err := result.Key()
		if err != nil {
			return Result{}, err
		}

		targetID := result.TargetID()
		status := result.Status()

		executionResult, err := result.ExecutionResult()
		if err != nil {
			return Result{}, err
		}

		execution := ResultExecution{
			Key: key,
			TargetID: targetID,
			Status: ExecutionStatus(status),
			ExecutionResult: executionResult,
		}

		executions = append(executions, execution)
	}

	resultMsg.Executions = executions

	return resultMsg, nil 
}