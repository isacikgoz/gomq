package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/isacikgoz/gomq/messaging"
)

var (
	udpClients map[string]*messaging.Client
)

func main() {
	udpClients = make(map[string]*messaging.Client)

	udpAddr := &net.UDPAddr{
		Port: 12345,
		IP:   net.ParseIP("127.0.0.1"),
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, syscall.SIGTERM, os.Interrupt)

	go listenOnUDPConnection(ctx, udpConn)

	fmt.Fprintf(os.Stdout, "started.\n")

	for {
		select {
		case <-ctx.Done():
			break
		case <-interrupt:
			cancel()
			return
		}
	}

}

func listenOnUDPConnection(ctx context.Context, udpConn *net.UDPConn) {
	var buffer [2048]byte

	for {
		select {
		case <-ctx.Done():
			return
		default:
			length, remoteAddr, err := udpConn.ReadFromUDP(buffer[0:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not read from UDP: %v\n", err)
				os.Exit(1)
			}

			// Check if we've seen UDP packets from this address before - if so, reuse
			// existing client object
			client, ok := udpClients[remoteAddr.String()]
			if !ok {
				fmt.Fprintf(os.Stdout, "new UDP client discovered!\n")
				writer := messaging.NewUDPWriter(udpConn, remoteAddr)
				bufferedWriter := bufio.NewWriter(writer)

				client = messaging.NewClient("name goes here", bufferedWriter, nil)
				udpClients[remoteAddr.String()] = client
			} else {
				fmt.Fprintf(os.Stdout, "Found UDP client!\n")
			}

			// Log the number of bytes received
			fmt.Fprintf(os.Stdout, "bytes received from UDP: %d, text: %s\n", length, string(buffer[:length]))

			// TODO: Parse message, and check if we're expecting a message
			commandTokens := strings.Fields(string(buffer[:length]))
			var message []byte
			if commandTokens[0] == "pub" {
				// Use bytes.Equal until Go1.7 (https://github.com/golang/go/issues/14302)
				for {
					var err error
					length, _, err := udpConn.ReadFromUDP(buffer[0:])

					if err != nil {
						return
					}

					// TODO: Is this cross platform? Needs testing
					if !bytes.Equal(buffer[:length], []byte{'.', '\r', '\n'}) {
						message = append(message, buffer[:length]...)
					} else {
						break
					}
				}
				fmt.Fprintf(os.Stdout, "read %d bytes from %s: %s\n", length, remoteAddr, string(buffer[:length]))
			}
		}
	}
}
