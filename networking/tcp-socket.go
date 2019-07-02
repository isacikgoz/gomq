package networking

import (
	"bufio"
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
func (l *TCPSocket) Listen() (*api.AnnotatedMessage, *messaging.Client, error) {
	buf := make([]byte, 16384)
	reader := l.client.Reader.(*bufio.Reader)
	length, err := reader.Read(buf)
	if err != nil {
		return nil, l.client, fmt.Errorf("could not read TCP: %v", err)
	}
	// Parse message
	var inc api.AnnotatedMessage
	json.Unmarshal(buf[:length], &inc)
	return &inc, l.client, nil
}
