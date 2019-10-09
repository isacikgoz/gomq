package messaging

import (
	"io"
)

// Client reads/writes from the given reader/writer
type Client struct {
	Name         string
	Writer       io.Writer
	Reader       io.Reader
	Closed       *chan bool
	AckRequested bool
}

// NewClient creates a new client
func NewClient(name string, writer io.Writer, reader io.Reader) *Client {
	close := make(chan bool)
	return &Client{
		Name:   name,
		Writer: writer,
		Reader: reader,
		Closed: &close,
	}
}
