package networking

import (
	"bufio"
	"context"
	"fmt"
	"net"

	"github.com/isacikgoz/gomq/internal/messaging"
)

// TCPListener listens incoming TCP connections
type TCPListener struct {
	listener *net.TCPListener
	addr     *net.TCPAddr

	clients map[string]*messaging.Client
	conns   []*TCPSocket
}

// NewTCPListener creates a new TCP connection handler
func NewTCPListener(ip string, port int, clients map[string]*messaging.Client) (*TCPListener, error) {
	TCPAddr := &net.TCPAddr{
		Port: port,
		IP:   net.ParseIP(ip),
	}
	l, err := net.ListenTCP("tcp", TCPAddr)
	if err != nil {
		return nil, fmt.Errorf("could not create TCP listener: %v", err)
	}

	listener := &TCPListener{
		listener: l,
		addr:     TCPAddr,
		clients:  clients,
	}
	return listener, nil
}

// AcceptConnection opens a TCP socket to communicate with a client
func (l *TCPListener) AcceptConnection(ctx context.Context) (*TCPSocket, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("an error occurred whilst opening a TCP socket for reading: %v", err)
	}
	connReader := bufio.NewReader(conn)
	connWriter := bufio.NewWriter(conn)
	closedChannel := make(chan bool)
	client := &messaging.Client{Name: "a TCP client",
		Writer: connWriter,
		Reader: connReader,
		Closed: &closedChannel}
	l.clients[l.addr.String()] = client

	socket := &TCPSocket{
		conn:   conn,
		client: client,
	}
	return socket, nil
}

// Close closes the connection.
func (l *TCPListener) Close() {
	for _, conn := range l.conns {
		conn.Close()
	}
}
