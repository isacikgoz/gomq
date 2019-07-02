package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/isacikgoz/gomq/networking"

	"github.com/isacikgoz/gomq/api"
	"github.com/isacikgoz/gomq/messaging"
)

var (
	udpClients map[string]*messaging.Client
	tcpClients map[string]*messaging.Client
	logger     io.Writer

	messages chan *api.AnnotatedMessage
)

func main() {
	udpClients = make(map[string]*messaging.Client)
	tcpClients = make(map[string]*messaging.Client)
	messages = make(chan *api.AnnotatedMessage, 2048)

	logger = os.Stdout

	UDPListener, err := networking.NewUDPListener("127.0.0.1", 12345, udpClients)
	defer UDPListener.Close()
	if err != nil {
		log.Fatal(err)
	}
	TCPListener, err := networking.NewTCPListener("127.0.0.1", 12345, tcpClients)
	defer TCPListener.Close()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, syscall.SIGTERM, os.Interrupt)

	wg := &sync.WaitGroup{}

	go listenOnUDPConnection(ctx, UDPListener)
	go listenOnTCPConnection(ctx, TCPListener, wg)

	if err := startServer(ctx, "8080"); err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(logger, "started.\n")
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Fprintln(logger, "context done.")
				quit <- struct{}{}
				return
			case <-interrupt:
				cancel()
				fmt.Fprintln(logger, "cancelling context...")
			case inc := <-messages:
				var msg api.BareMessage
				json.Unmarshal(inc.Payload, &msg)
				fmt.Fprintf(logger, "incoming command: %s, target: %s\n", inc.Command, inc.Target)
				fmt.Fprintf(logger, "incoming message: %s\n", msg.Message)
			}
		}
	}()
	<-quit
	wg.Wait()

}

func listenOnUDPConnection(ctx context.Context, listener *networking.UDPListener) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(logger, "gracefully shutting down")
			return
		default:
			inc, err := listener.Listen(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not listen UDP: %v\n", err)
				return
			}
			messages <- inc
		}
	}
}

func listenOnTCPConnection(ctx context.Context, listener *networking.TCPListener, wg *sync.WaitGroup) {
	defer wg.Done()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			socket, err := listener.AcceptConnection(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "an error occurred whilst opening a TCP socket for reading: %v\n", err)
			}

			go func(socket *networking.TCPSocket) {
				for {
					inc, err := socket.Listen()

					if err != nil {
						// Connection has been closed
						fmt.Fprintln(logger, "closed a connection")
						break
					}
					messages <- inc
				}
				fmt.Fprintf(logger, "a TCP connection was closed\n")
			}(socket)
		}
	}
}

func startServer(ctx context.Context, port string) error {
	ctx, cancel := context.WithCancel(ctx)
	server := &http.Server{Addr: ":" + port}

	http.HandleFunc("/", clientsHandler)

	go func() {
		if err := server.ListenAndServe(); err != nil &&
			!strings.Contains(err.Error(), "Server closed") {
			fmt.Fprintf(os.Stderr, "server stopped: %v\n", err)
			cancel()
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			if err := server.Shutdown(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "server shutdown error: %v\n", err)
			}
		}
	}()
	return nil
}

func clientsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	clients := make([]string, 0)

	for client := range udpClients {
		clients = append(clients, client)
	}

	json.NewEncoder(w).Encode(clients)
}
