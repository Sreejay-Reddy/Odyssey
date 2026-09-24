package socket

import (
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"

	"github.com/sreejay-reddy/odyssey/protocol/gen/go"
	"capnproto.org/go/capnp/v3"
)

func EncodeRegistry(r *registry.Registry) (*capnp.Message, error) {
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	root, err := protocol.NewRegistryMessage(seg)

	sdkID := []byte("testing")
	err = root.SetSdkID(sdkID)
	if err != nil {
		return nil, err
	}

	sessionID := []byte("testing")
	err = root.SetSessionID(sessionID)
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