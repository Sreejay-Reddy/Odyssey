package workers

import (
	"net"

	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
)

type Worker struct {
	WorkerID string
	Event    net.Conn
	Command	 net.Conn
	registry *registry.Registry
}

func (w *Worker) RunWorker(
	workerID string, 
	event net.Conn, 
	command net.Conn, 
	registry *registry.Registry) error {

		return nil

}