package socket

import(
	"net"
	"context"

	"capnproto.org/go/capnp/v3"
)

func RunWriter(ctx context.Context, conn net.Conn, send <-chan *capnp.Message) error {
	encoder := capnp.NewEncoder(conn)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg, ok := <-send:
			if !ok {
				return nil
			}

			if err := encoder.Encode(msg); err != nil {
				return err
			}
		}
	}
}