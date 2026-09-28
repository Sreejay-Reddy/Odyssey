package socket

import (
	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"

	"capnproto.org/go/capnp/v3"
	"github.com/sreejay-reddy/odyssey/protocol/gen/go"
)

func EncodeRegistry(r *registry.Registry, state configutil.State) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	root, err := protocol.NewRootRegistryMessage(seg)
	if err != nil {
		return nil, err
	}

	err = root.SetSdkID(state.SDKID[:])
	if err != nil {
		return nil, err
	}

	err = root.SetSessionID(state.SessionID[:])
	if err != nil {
		return nil, err
	}

	targets := r.All()

	list, err := root.NewTargets(int32(len(targets)))

	for i, target := range targets {
		registered := list.At(i)

		err = registered.SetFunctionName(target.FunctionName)
		if err != nil {
			return nil, err
		}

		err = registered.SetName(target.Target)
		if err != nil {
			return nil, err
		}

		registered.SetTargetID(target.TargetID)
		registered.SetTtlMS(uint32(target.TTLMS))
	}

	return msg, nil
}

func EncodeResult(result Result) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	root, err := protocol.NewRootResultMessage(seg)
	if err != nil {
		return nil, err
	}

	root.SetProtocolVersion(result.Version)
	root.SetSdkID(result.SDKID[:])
	root.SetSessionID(result.SessionID[:])

	list, err := root.NewExecutions(int32(len(result.Executions)))
	if err != nil {
		return nil, err
	}

	for i, exec := range result.Executions {
		msgExec := list.At(i)

		err := msgExec.SetKey(exec.Key)
		if err != nil {
			return nil, err
		}

		msgExec.SetTargetID(exec.TargetID)
		msgExec.SetStatus(protocol.ExecutionStatus((exec.Status)))

		err = msgExec.SetExecutionResult(exec.ExecutionResult)
		if err != nil {
			return nil, err
		}

	}

	return msg, nil
}