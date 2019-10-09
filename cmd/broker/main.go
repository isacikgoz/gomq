package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/isacikgoz/gomq/api"
	"github.com/isacikgoz/gomq/internal/messaging"
	"github.com/isacikgoz/gomq/internal/networking"
	"github.com/isacikgoz/gomq/internal/server"
)

var (
	logger     io.Writer
	messages   chan *IncomingMessage
	dispatcher *messaging.Dispatcher
)

// IncomingMessage holds a message with sender information
type IncomingMessage struct {
	message *api.AnnotatedMessage
	sender  *messaging.Client
}

func main() {
	udpClients := make(map[string]*messaging.Client)
	tcpClients := make(map[string]*messaging.Client)

	messages = make(chan *IncomingMessage, 2048)

	dispatcher = messaging.NewDispatcher()

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

	go listenUDP(ctx, UDPListener)
	go listenTCP(ctx, TCPListener)

	backend := server.NewBackend(udpClients, tcpClients)

	if err := backend.StartServer(ctx, "8080"); err != nil {
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
			case rm := <-messages:
				msg := rm.message
				client := rm.sender

				switch msg.Command {
				case "PUB":
					dispatcher.Dispatch(msg, client)
				case "SUB":
					dispatcher.Subscribe(msg.Target, client)
				case "UNS":
					dispatcher.Unsubscribe(msg.Target, client)
				default:
					// send client that msg is unrecognized
				}
			}
		}
	}()
	<-quit
}

func listenUDP(ctx context.Context, listener *networking.UDPListener) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(logger, "gracefully shutting down")
			return
		default:
			inc, client, err := listener.Listen(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not listen UDP: %v\n", err)
				return
			}
			messages <- &IncomingMessage{inc, client}
		}
	}
}

func listenTCP(ctx context.Context, listener *networking.TCPListener) {
	wg := &sync.WaitGroup{}
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

			wg.Add(1)
			go func(socket *networking.TCPSocket) {
				defer wg.Done()
				for {
					inc, client, err := socket.Listen()

					if err != nil {
						// Connection has been closed
						break
					}
					messages <- &IncomingMessage{inc, client}
				}
				fmt.Fprintf(logger, "a TCP connection was closed\n")
			}(socket)
		}
		wg.Wait()
	}
}
