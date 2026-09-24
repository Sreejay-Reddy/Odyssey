package server

import (
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/socket"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"

	"capnproto.org/go/capnp/v3"
)

func Start(r *registry.Registry) error {
	ackconn, err := socket.CreateAckSocket()
	if err != nil {
		return err
	}
	ackencoder := capnp.NewEncoder(ackconn)

	registryMsg, err := socket.EncodeRegistry(r)
	if err != nil {
		return err
	}

	err = ackencoder.Encode(registryMsg)
	if err != nil {
		return err
	}

	return nil 
}

