package socket

import(
	"net"
	"fmt"
	"time"
	"context"
	"path/filepath"
)

const SocketDir = "/tmp/odyssey"
const AckPath   = SocketDir + "/ack.sock"

func createSocket(ctx context.Context, path string) (net.Conn, error) {
	var dialer net.Dialer

	for {
		conn, err := dialer.DialContext(ctx, "unix", path)
		if err == nil {
			return conn, nil
		}

		timer := time.NewTimer(time.Second)

		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()

		case <-timer.C:
		}
	}
}

func CreateWorkers(ctx context.Context, workers int) ([]net.Conn, []net.Conn, error) {
	sockets := make([]net.Conn, 0, workers)
	eventSockets := make([]net.Conn, 0, workers)

	for worker := 0; worker < workers; worker++ {
		workerID := fmt.Sprintf("worker-%d", worker)
		resultWorkerID := fmt.Sprintf("result-%d", worker)
		path := filepath.Join(SocketDir, workerID+".sock")
		resultPath := filepath.Join(SocketDir, resultWorkerID+".sock")

		conn, err := createSocket(ctx, path)
		if err != nil {
			for _, socket := range sockets {
				socket.Close()
			}

			return nil, nil, fmt.Errorf(
				"failed to connect worker socket %s: %w",
				workerID,
				err,
			)
		}

		eventconn, err := createSocket(ctx, resultPath)
		if err != nil {
			for _, eventsocket := range eventSockets {
				eventsocket.Close()
			}

			return nil, nil, fmt.Errorf(
				"failed to connect worker socket %s: %w",
				workerID,
				err,
			)
		}

		sockets = append(sockets, conn)
		eventSockets = append(eventSockets, eventconn)
	}

	return sockets, eventSockets, nil
}

func CreateAckSocket(ctx context.Context) (net.Conn, error) {
	return createSocket(ctx, AckPath)
}
