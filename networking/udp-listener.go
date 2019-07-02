package networking

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/isacikgoz/gomq/api"
	"github.com/isacikgoz/gomq/messaging"
)

// UDPListener listens incoming messages from UDP
type UDPListener struct {
	conn *net.UDPConn
	addr *net.UDPAddr

	clients map[string]*messaging.Client
}

// NewUDPListener creates a new UDP listener from given addresses
func NewUDPListener(ip string, port int, clients map[string]*messaging.Client) (*UDPListener, error) {
	UDPAddr := &net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(ip),
	}
	UDPConn, err := net.ListenUDP("udp", UDPAddr)
	if err != nil {
		return nil, fmt.Errorf("could not create UDP listener: %v", err)
	}

	listener := &UDPListener{
		conn:    UDPConn,
		addr:    UDPAddr,
		clients: clients,
	}
	return listener, nil
}

// Close closes the connection.
func (l *UDPListener) Close() {
	l.conn.Close()
}

// Listen listens the incoming UDP data and parses it
func (l *UDPListener) Listen(ctx context.Context) (*api.AnnotatedMessage, error) {
	var buffer [16384]byte
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		length, remoteAddr, err := l.conn.ReadFromUDP(buffer[0:])
		if err != nil {
			return nil, fmt.Errorf("could not read UDP: %v", err)
		}

		// Check if we've seen UDP packets from this address before - if so, reuse
		// existing client object
		client, ok := l.clients[remoteAddr.String()]
		if !ok {
			writer := messaging.NewUDPWriter(l.conn, remoteAddr)
			bufferedWriter := bufio.NewWriter(writer)

			client = messaging.NewClient("name goes here", bufferedWriter, nil)
			l.clients[remoteAddr.String()] = client
		}

		// Parse message
		var inc api.AnnotatedMessage
		json.Unmarshal(buffer[:length], &inc)
		return &inc, nil
	}
}
