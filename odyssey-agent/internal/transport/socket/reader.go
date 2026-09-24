package socket

import (
	"context"
	"net"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/batcher"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"

	"capnproto.org/go/capnp/v3"
)

func RunEventReader(ctx context.Context, 
	resultconn net.Conn, 
	batchclient *batcher.Batcher,
	r *registry.Registry) (error) {
		decoder := capnp.NewDecoder(resultconn)
		for {
			select {
				case <-ctx.Done():
					return ctx.Err()
				default:
			}

			msg, err := decoder.Decode()
			if err != nil {
				return err
			}

			result, err := DecodeResult(msg)
			if err != nil {
				return err
			}

			executions := make([]storage.Execution, 0, len(result.Executions))
			failedexecutions := make([]storage.Execution, 0, len(result.Executions))

			for _, execution := range result.Executions {

				registered, err := r.GetByID(execution.TargetID)
				if err != nil {
					return err
				}

				executed := storage.Execution {
					Key: execution.Key,
					Target: registered.Target,
					ExecutionResult: execution.ExecutionResult,
				}

				if (execution.Status == StatusSuccess) {
					executions = append(executions, executed)
				}else {
					failedexecutions = append(failedexecutions, executed)
				}
			}

			go batchclient.BatchComplete(ctx, executions)
		}
}