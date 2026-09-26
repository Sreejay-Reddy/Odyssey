package server

import (
	"context"
	"fmt"
	"net"

	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/socket"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/workers"

	"capnproto.org/go/capnp/v3"
)

func Start(
	ctx context.Context,
	r *registry.Registry,
	cfg configutil.Config,
	state configutil.State) error {
	ackconn, err := socket.CreateAckSocket()
	if err != nil {
		return err
	}
	ackencoder := capnp.NewEncoder(ackconn)

	registryMsg, err := socket.EncodeRegistry(r, state)
	if err != nil {
		return err
	}

	err = ackencoder.Encode(registryMsg)
	if err != nil {
		return err
	}

	commandConns, eventConns, err := socket.CreateWorkers(cfg.Agent.SDK.Workers)
	if err != nil {
		return err
	}

	events := make([]chan workers.Response, cfg.Agent.SDK.Workers)
	errCh := make(chan error, cfg.Agent.SDK.Workers)

	for i, commandConn := range commandConns {
		event := make(chan workers.Response, cfg.Agent.SDK.BatchSize)

		go func(commandConn net.Conn, event chan workers.Response){	
			err := workers.RunWorker(ctx, commandConn, r, event)
			if err != nil{
				errCh <- fmt.Errorf("execution worker %d: %w", i, err)
			}
		}(commandConn, event)

		events[i] = event
	}

	for i, eventConn := range eventConns {
		event := events[i]

		go func(eventConn net.Conn, cfg configutil.Config, state configutil.State, event chan workers.Response){
			err := workers.RunReader(ctx, eventConn, cfg, state, event)
			if err != nil{
				errCh <- fmt.Errorf("reader worker %d: %w", i, err)
			}
		}(eventConn, cfg, state, event)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()

	case err := <-errCh:
		return err
	}
}

