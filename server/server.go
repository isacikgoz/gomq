package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/isacikgoz/gomq/messaging"
)

// Backend is the REST API for query the clients
type Backend struct {
	udpClients map[string]*messaging.Client
	tcpClients map[string]*messaging.Client
}

// NewBackend returns a new backend struct
func NewBackend(udps, tcps map[string]*messaging.Client) *Backend {
	return &Backend{
		udpClients: udps,
		tcpClients: tcps,
	}
}

// StartServer starts an http server
func (b *Backend) StartServer(ctx context.Context, port string) error {
	ctx, cancel := context.WithCancel(ctx)
	server := &http.Server{Addr: ":" + port}

	http.HandleFunc("/", b.clientsHandler)

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

func (b *Backend) clientsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	clients := make([]string, 0)

	for client := range b.udpClients {
		clients = append(clients, "udp: "+client)
	}
	for client := range b.tcpClients {
		clients = append(clients, "tcp: "+client)
	}

	json.NewEncoder(w).Encode(clients)
}
