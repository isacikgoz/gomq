package networking

import (
	"net"
)

// Writer is a simple implementation of the writer interface, for sending
// UDP messages
type Writer struct {
	addr *net.UDPAddr
	conn *net.UDPConn
}

// NewUDPWriter returns a new UDP Writer
func NewUDPWriter(givenConnection *net.UDPConn, givenAddress *net.UDPAddr) *Writer {
	return &Writer{
		addr: givenAddress,
		conn: givenConnection,
	}
}

// Write is an implementation of the standard io.Writer interface
func (w *Writer) Write(givenBytes []byte) (int, error) {
	return w.conn.WriteToUDP(givenBytes, w.addr)
}
