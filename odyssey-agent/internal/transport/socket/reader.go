package socket

import (
	"context"
	"net"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/batcher"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
)

func RunEventReader(ctx context.Context, 
	resultconn net.Conn, 
	batchclient *batcher.Batcher,
	r *registry.Registry) (error) {
		for {
			select {
				case <-ctx.Done():
					return ctx.Err()
				default:
			}

			// msg, err := ReadMessage(resultconn)
			// if err != nil {
			// 	return err
			// }

			var b []byte

			result, err := DecodeResult(resultconn, b)
			if err != nil {
				return err
			}

			executions := make([]storage.Execution, 0, len(result.Executions))

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

				executions = append(executions, executed)
			}

			go batchclient.BatchComplete(ctx, executions)
		}
}