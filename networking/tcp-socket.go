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
func (l *TCPSocket) Listen() (*api.AnnotatedMessage, error) {
	buf := make([]byte, 4096)
	reader := l.client.Reader.(*bufio.Reader)
	length, err := reader.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("could not read TCP: %v", err)
	}
	// Parse message
	var inc api.AnnotatedMessage
	json.Unmarshal(buf[:length], &inc)
	return &inc, nil
}
