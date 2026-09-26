package workers

import (
	"context"
	"time"
	"net"

	"capnproto.org/go/capnp/v3"
	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/socket"
)

func RunReader(
	ctx context.Context,
	event net.Conn,
	cfg configutil.Config,
	state configutil.State,
	events <-chan Response,
) error {
	encoder := capnp.NewEncoder(event)

	batchSize := cfg.Agent.SDK.BatchSize
	flushInterval := 200 * time.Millisecond

	results := make([]socket.ResultExecution, 0, batchSize)

	timer := time.NewTimer(flushInterval)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	flush := func() error {
		if len(results) == 0 {
			return nil
		}

		result := socket.Result{
			Version:    socket.ProtocolVersion,
			SDKID:      state.SDKID,
			SessionID:  state.SessionID,
			Executions: results,
		}

		msg, err := socket.EncodeResult(result)
		if err != nil {
			return err
		}

		if err := encoder.Encode(msg); err != nil {
			return err
		}

		results = make([]socket.ResultExecution, 0, batchSize)

		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case response, ok := <-events:
			if !ok {
				return flush()
			}

			results = append(results, socket.ResultExecution{
				Key:             response.Key,
				TargetID:        response.TargetID,
				ExecutionResult: response.Response,
				Status:          response.Status,
			})

			if len(results) == 1 {
				timer.Reset(flushInterval)
			}

			if len(results) >= batchSize {
				if err := flush(); err != nil {
					return err
				}

				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			}

		case <-timer.C:
			if err := flush(); err != nil {
				return err
			}
		}
	}
}