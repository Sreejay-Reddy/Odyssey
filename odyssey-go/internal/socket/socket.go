package socket

import (
	"os"
	"fmt"
	"net"
	"time"
	"context"
	"path/filepath"

)

const SocketDir = "/tmp/odyssey"
const AckPath = "/tmp/odyssey-ack.sock"

func createListener(ctx context.Context, path string) (net.Listener, error) {
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return nil, err
    }

    for {
        listener, err := net.Listen("unix", path)
        if err == nil {
            return listener, nil
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

func CreateWorkers(ctx context.Context, workers int) ([]net.Conn, []net.Conn, error){
	sockets := make([]net.Conn, 0, workers)
	eventSockets := make([]net.Conn, 0, workers)

	listeners := make([]net.Listener, 0, workers)
    eventListeners := make([]net.Listener, 0, workers)

	for i:=0; i<workers; i++ {
		workerID := fmt.Sprintf("worker-%d", i)
		resultWorkerID := fmt.Sprintf("result-%d", i)
		path := filepath.Join(SocketDir, workerID+".sock")
		resultPath := filepath.Join(SocketDir, resultWorkerID+".sock")

		listener, err := createListener(ctx, path)
		if err != nil {
			return nil, nil, err
		}

		eventListener, err := createListener(ctx, resultPath)
		if err != nil {
			return nil, nil, err
		}

		eventListeners = append(eventListeners, eventListener)
		listeners = append(listeners, listener)
	}

	for i:=0; i<workers; i++ {
		conn, err := listeners[i].Accept()
		if err != nil {
			return nil, nil, err
		}
		listeners[i].Close()

		eventConn, err := eventListeners[i].Accept()
		if err != nil {
			return nil, nil, err
		}
		eventListeners[i].Close()

		sockets = append(sockets, conn)
		eventSockets = append(eventSockets, eventConn)
	}

	return sockets, eventSockets, nil
}

func CreateAckSocket()(net.Conn, error){
	listener, err := net.Listen("unix", AckPath)
	if err != nil {
		return nil, err
	}

	conn, err := listener.Accept()
	if err != nil {
		return nil, err
	}

	return conn, nil
} 