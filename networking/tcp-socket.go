package networking

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/isacikgoz/gomq/messaging"

	"github.com/isacikgoz/gomq/api"
)

// TCPSocket reads from single tcp connection
type TCPSocket struct {
	conn   net.Conn
	client *messaging.Client
}

// Close closes the connection
func (l *TCPSocket) Close() {
	l.conn.Close()
}

// Listen listens messages from a TCP socket
func (l *TCPSocket) Listen() (*api.AnnotatedMessage, error) {
	var buffer []byte
	length, err := l.client.Reader.Read(buffer)
	if err != nil {
		// Connection has been closed
		*l.client.Closed <- true // TODO: This is blocking - shouldn't be
		return nil, fmt.Errorf("could not read from socket: %v", err)
	}

	// Parse message
	var inc api.AnnotatedMessage
	json.Unmarshal(buffer[:length], &inc)
	return &inc, nil
}
