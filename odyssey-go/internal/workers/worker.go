package workers

import (
	"context"
	"net"

	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/execute"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/socket"

	"capnproto.org/go/capnp/v3"
)

func RunWorker(
	ctx context.Context,
	command net.Conn, 
	r *registry.Registry,
	events chan<- Response) error {

	decoder := capnp.NewDecoder(command)
	worker := execute.Execution{
		Registry: r,
	}

	for {
		select{
			case<-ctx.Done():
				return ctx.Err()
		
			default:
		}
		
		msg, err := decoder.Decode()
		if err != nil {
			return err
		}

		commandMsg, err := socket.DecodeCommand(msg)
		if err != nil {
			return err
		}

		for _, execution := range commandMsg.Executions {
			go func (execution socket.Execution){
				response, err := worker.Execute(
					ctx, 
					execution.Key, 
					execution.TargetID, 
					execution.Input,
				)

				if err != nil {
					res := Response{
						Key: execution.Key,
						TargetID: execution.TargetID,
						err: err,
						Status: socket.StatusFailed,
					}

					events <- res
				}

				if err == nil {
					res := Response{
						Key: execution.Key,
						TargetID: execution.TargetID,
						err: nil,
						Status: socket.StatusSuccess,
						Response: response,
					}

					events <- res
				}
			}(execution)

		}
	}
}