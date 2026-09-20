package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/batcher"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/config"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/scheduler"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/server"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage/postgres"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/transport/socket"


	"gopkg.in/yaml.v3"
)

func runBatchLoop (ctx context.Context, 
	worker scheduler.Worker, 
	r *registry.Registry, 
	batchclient *batcher.Batcher) (error) {
	for {

		batch, err := batchclient.Next(ctx, worker.ID)
		if err != nil {
			return  err
		}

		msg, err := socket.EncodeMessage(r, batch)
		if err != nil {
			return err
		}

		encoded, err := socket.FrameMessage(msg)
		if err != nil {
			return err
		}

		worker.Send<- encoded

	}
}

func run () (error) {
    ctx, stop := signal.NotifyContext(
        context.Background(),
        os.Interrupt,
        syscall.SIGTERM,
    )
    defer stop()

	dbURL, exists := os.LookupEnv("DATABASE_URL")
	if exists != true{
		return errors.New("DATABASE_URL not set")
	}

	slog.Info("Database", "db_url", dbURL)

	path, err := config.FindConfig()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfigYAML(path)
	if err != nil{
		return err
	}

	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		slog.Error("failed to marshal config", "error", err)
		return nil
	}

	slog.Info("Odyssey.yaml config")
	fmt.Print(string(yamlData))

	ackconn, err := socket.CreateAckSocket(ctx)
	if err != nil {
		return err
	}

	slog.Info("connected ack socket")

	msg, err := socket.ReadMessage(ackconn)
	if err != nil {
		return err
	}

	r := registry.New()

	err = socket.DecodeRegistry(msg, r)
	if err != nil {
		return err
	}

	commandConns, eventConns, err := socket.CreateWorkers(ctx, cfg.Agent.SDK.Workers)
	if err != nil { 
		return err
	}

	slog.Info("Starting Workers....", "Workers", cfg.Agent.SDK.Workers)

	sends := make([]chan<- []byte, len(commandConns))

	for i, conn := range commandConns {
		send := make(chan []byte, 64)

		go socket.RunWriter(ctx, conn, send)

		sends[i] = send
	}

	sch := scheduler.NewScheduler(commandConns, eventConns, sends)

	pool, err := postgres.NewPool(ctx, cfg.Agent.Postgres, dbURL)
	if err != nil {
		return err
	}

	writer := postgres.New(pool, cfg)

	if cfg.Agent.SDK.BatchSize == 0 {
		cfg.Agent.SDK.BatchSize = 128
	}

	batchclient := batcher.New(writer, r, cfg.Agent.SDK.BatchSize, time.Duration(1)*time.Second)

	for _, worker := range sch.Workers() {
		go func(worker scheduler.Worker){
		err := runBatchLoop(ctx, worker, r, batchclient)
			if err != nil {
				slog.Error("batch loop failed", "error", err)
				stop()
			}
		}(worker)
	}

	for _, eventConn := range eventConns {
		go func(conn net.Conn) {
			err := socket.RunEventReader(ctx, conn, batchclient, r)
			if err != nil {
				slog.Error("Event Reader failed", "error", err)
				stop()
			}
		}(eventConn)
	}

	s := server.New(":8080")

	go func(){
		err := s.Start()
		if err != nil {
			slog.Error("server failed", "error", err)
			s.Shutdown(ctx)
		}
	}()

	<-ctx.Done()

	slog.Info("shutting down agent")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err = s.Shutdown(shutdownCtx)
	if err != nil {
		return err
	}

	pool.Close()

	for _, conn := range commandConns {
		conn.Close()
	}

	for _, conn := range eventConns {
		conn.Close()
	}

	slog.Info("agent shutdown gracefully")

	return nil
}

func main() {
	err := run() 
	if err != nil {
		panic(err)
	}
}