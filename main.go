package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/isacikgoz/gomq/api"
	"github.com/isacikgoz/gomq/messaging"
)

var (
	udpClients map[string]*messaging.Client
)

func main() {
	udpClients = make(map[string]*messaging.Client)

	udpAddr := &net.UDPAddr{
		Port: 12345,
		IP:   net.ParseIP("127.0.0.1"),
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	defer udpConn.Close()

	if err != nil {
		log.Fatal(err)
	}

	// Open TCP socket
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":12345")

	if err != nil {
		panic(err)
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	defer tcpListener.Close()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, syscall.SIGTERM, os.Interrupt)

	go listenOnUDPConnection(ctx, udpConn)

	if err := startServer(ctx, "8080"); err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(os.Stdout, "started.\n")
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Fprintln(os.Stdout, "context done.")
				quit <- struct{}{}
				return
			case <-interrupt:
				cancel()
				fmt.Fprintln(os.Stdout, "cancelling context...")
			}
		}
	}()
	<-quit
}

func listenOnUDPConnection(ctx context.Context, udpConn *net.UDPConn) {
	var buffer [2048]byte
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stdout, "gracefully shutting down")
			return
		default:
			length, remoteAddr, err := udpConn.ReadFromUDP(buffer[0:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not read from UDP: %v\n", err)
				os.Exit(1)
			}

			// Check if we've seen UDP packets from this address before - if so, reuse
			// existing client object
			client, ok := udpClients[remoteAddr.String()]
			if !ok {
				fmt.Fprintf(os.Stdout, "new UDP client discovered!\n")
				writer := messaging.NewUDPWriter(udpConn, remoteAddr)
				bufferedWriter := bufio.NewWriter(writer)

				client = messaging.NewClient("name goes here", bufferedWriter, nil)
				udpClients[remoteAddr.String()] = client
			} else {
				fmt.Fprintf(os.Stdout, "Found UDP client!\n")
			}

			// Log the number of bytes received
			fmt.Fprintf(os.Stdout, "bytes received from UDP: %d\n", length)

			// Parse message
			var inc api.AnnotatedMessage
			json.Unmarshal(buffer[:length], &inc)
			fmt.Fprintf(os.Stdout, "incoming command: %s, target: %s\n", inc.Command, inc.Target)
			var msg api.BareMessage
			json.Unmarshal(inc.Payload, &msg)
			fmt.Fprintf(os.Stdout, "incoming message: %s\n", msg.Message)
		}
	}
}

func startServer(ctx context.Context, port string) error {
	ctx, cancel := context.WithCancel(ctx)
	server := &http.Server{Addr: ":" + port}

	http.HandleFunc("/", clientsHandler)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "server stopped: %v", err)
			cancel()
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			if err := server.Shutdown(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "server stopped: %v", err)
				os.Exit(1)
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
