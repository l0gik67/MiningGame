package main

import (
	gamehttp "MiningGame/http"
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	address := flag.String("addr", ":8080", "Адрес HTTP-сервера")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	listener, err := net.Listen("tcp", *address)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	log.Printf("MiningGame слушает %s", listener.Addr())
	if err := serve(ctx, listener, gamehttp.NewGameHandler()); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func serve(ctx context.Context, listener net.Listener, handler *gamehttp.GameHandler) error {
	defer handler.Close()
	server := &http.Server{
		Handler:           handler.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	result := make(chan error, 1)
	go func() {
		result <- server.Serve(listener)
	}()

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	case <-handler.Done():
	}

	handler.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		<-result
		return err
	}
	if err := <-result; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
