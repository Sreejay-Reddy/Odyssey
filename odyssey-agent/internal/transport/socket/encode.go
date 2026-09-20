package socket

import (
	"encoding/binary"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/batcher"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"

	"github.com/sreejay-reddy/odyssey/protocol/gen/go"
	"capnproto.org/go/capnp/v3"
)

func EncodeMessage(r *registry.Registry, batch batcher.Batch) (*capnp.Message, error) {
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

func FrameMessage(msg *capnp.Message) ([]byte, error) {
	payload, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 4+len(payload))

	binary.BigEndian.PutUint32(
		buf[:4],
		uint32(len(payload)),
	)

	copy(buf[4:], payload)

	return buf, nil
}