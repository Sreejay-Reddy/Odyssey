package workers

import (
	"context"
	"net"

	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/execute"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/socket"

	"capnproto.org/go/capnp/v3"
)

type Worker struct {
	WorkerID string
	Event    net.Conn
	Command	 net.Conn
	registry *registry.Registry
}

func (w *Worker) RunWorker(
	ctx context.Context,
	workerID string, 
	event net.Conn, 
	command net.Conn, 
	r *registry.Registry) error {

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
				_, _ = worker.Execute(ctx, 
					execution.Key, 
					execution.TargetID, 
					execution.Input,
				)
			}(execution)

		}
	}
}