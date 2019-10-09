package messaging

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/isacikgoz/gomq/api"
)

// Dispatcher routes the messages
type Dispatcher struct {
	queues    map[string]chan *api.AnnotatedMessage
	listeners map[string][]*Client
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewDispatcher generates a new Dispatcher
func NewDispatcher() *Dispatcher {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	return &Dispatcher{
		queues:    make(map[string](chan *api.AnnotatedMessage)),
		listeners: make(map[string][]*Client),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Dispatch routes message to the channel
func (d *Dispatcher) Dispatch(msg *api.AnnotatedMessage, c *Client) error {
	queue, ok := d.queues[msg.Target]
	if !ok {
		return nil // maybe an error? matter of taste atm
	}
	queue <- msg
	return nil
}

// Subscribe adds a listener to the topic of the events on the Queue
func (d *Dispatcher) Subscribe(topic string, c *Client) error {

	d.listeners[topic] = append(d.listeners[topic], c)

	_, ok := d.queues[topic]
	if !ok {
		d.queues[topic] = make(chan *api.AnnotatedMessage)
		go d.collate(topic) // start routine for each context ie. topic
	}
	return nil
}

// Unsubscribe removes a listener from the topic
func (d *Dispatcher) Unsubscribe(topic string, c *Client) error {

	l := d.listeners[topic]

	for i, listener := range l {
		if listener.Name == c.Name {
			copy(l[i:], l[i+1:])
			l[len(l)-1] = nil // or the zero value of T
			l = l[:len(l)-1]
			d.listeners[topic] = l
			break
		}
	}

	return nil
}

func (d *Dispatcher) collate(topic string) {
	ctx, cancel := context.WithCancel(d.ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-d.queues[topic]:
			if err := d.publish(topic, msg); err != nil {
				log.Printf("message could not send: %v", err)
			}
		}
	}
}

func (d *Dispatcher) publish(topic string, msg *api.AnnotatedMessage) error {
	// maybe a mutex here?
	// find the listeners for this context
	listeners, ok := d.listeners[topic]
	if !ok {
		return nil // no listeners
	}
	// publish data to the listeners
	for i := range listeners {

		// don't check err if not interested in if the msg is sent
		b, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("could not publish: %v", err)
		}
		writer := listeners[i].Writer.(*bufio.Writer)
		if _, err := writer.Write(b); err != nil {
			return fmt.Errorf("could not write to connection: %v", err)
		}
		writer.Flush()
	}
	return nil
}

// Close closes the goroutines of the dispatcher.
func (d *Dispatcher) Close() {
	d.cancel()
}
