package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/isacikgoz/gomq/api"
)

// BareMessage just does fine
type BareMessage struct {
	Message string
}

var (
	inc       = make(chan api.AnnotatedMessage)
	out       = make(chan api.AnnotatedMessage)
	interrupt = make(chan os.Signal)
)

func main() {

	protocol := flag.String("protocol", "udp", "the protocol to connect to the broker.")
	topic := flag.String("topic", "default", "topic to be subscribed.")
	name := flag.String("name", "", "user name.")

	flag.Parse()
	fmt.Printf("the protocol is %q and subscribed to topic %q as %q\n", *protocol, *topic, *name)

	conn, err := net.Dial(*protocol, "127.0.0.1:12345")
	defer conn.Close()
	errorf(err)

	errorf(subscribe(conn, *name, *topic))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	signal.Notify(interrupt, syscall.SIGTERM, os.Interrupt)

	go listenFromBroker(conn, inc)
	go listenFromUser(os.Stdin, out, *topic)
	go selectMessages(ctx, conn, inc, out)

	<-interrupt
}

func errorf(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "some error %v\n", err)
		os.Exit(1)
	}
}

func selectMessages(ctx context.Context, conn io.Writer, inc, out chan api.AnnotatedMessage) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("goodbye!")
			return
		case msg := <-inc:
			var bare BareMessage
			json.Unmarshal(msg.Payload, &bare)
			log.Printf("%s\n", bare.Message)
		case msg := <-out:
			b, err := json.Marshal(msg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error wrapping message: %v", err)
				os.Exit(1)
			}
			conn.Write(b)
		}
	}
}

func listenFromUser(rd io.Reader, ch chan api.AnnotatedMessage, topic string) {
	s := bufio.NewScanner(rd)

	for s.Scan() {
		pl := BareMessage{
			Message: s.Text(),
		}
		plData, err := json.Marshal(pl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error wrapping message: %v", err)
			continue
		}

		msg := api.AnnotatedMessage{
			Command: "PUB",
			Target:  topic,
			Payload: plData,
		}
		ch <- msg
	}
}

func listenFromBroker(rd io.Reader, ch chan api.AnnotatedMessage) {
	for {
		p := make([]byte, 16384)
		n, err := bufio.NewReader(rd).Read(p)
		if err != nil {
			break // probably network connection closed
		}

		var msg api.AnnotatedMessage
		if err := json.Unmarshal(p[:n], &msg); err != nil {
			continue // maybe break
		}
		ch <- msg
	}
}

func subscribe(rd io.Writer, name, topic string) error {
	sub := &api.AnnotatedMessage{
		Sender:  name,
		Command: "SUB",
		Target:  topic,
	}

	b, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("could not send subscribe message: %v", err)
	}
	rd.Write(b)
	return nil
}
