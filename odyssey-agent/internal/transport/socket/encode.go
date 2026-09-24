package socket

import (
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/batcher"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"

	"github.com/sreejay-reddy/odyssey/protocol/gen/go"
	"capnproto.org/go/capnp/v3"
)

func EncodeBatchMessage(r *registry.Registry, batch batcher.Batch) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	root, err := protocol.NewRootSubmitMessage(seg)
	if err != nil {
		return nil, err
	}

	root.SetProtocolVersion(ProtocolVersion)
	root.SetBatchID(batch.ID)

	list, err := root.NewExecutions(int32(len(batch.Executions)))
	if err != nil {
		return nil, err
	}

	for i, claim := range batch.Executions {
		registered, err := r.GetByName(claim.Target)
		if err != nil {
			return nil, err
		}

		exec := list.At(i)

		exec.SetKey(claim.Key)
		exec.SetTargetID(registered.TargetID)

		err = exec.SetInput(claim.Input)
		if err != nil {
			return nil, err
		}
	}

	return msg, nil
}
