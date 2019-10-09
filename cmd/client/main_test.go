package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/isacikgoz/gomq/api"
)

type mockReader struct {
	bytes []byte
}

func (mr *mockReader) Read(bytes []byte) (int, error) {
	message := &BareMessage{
		Message: "testing",
	}
	payload, _ := json.Marshal(message)
	mockMsg := &api.AnnotatedMessage{
		Command: "PUB",
		Target:  "topic_1",
		Payload: payload,
	}
	src, _ := json.Marshal(mockMsg)
	for i, b := range src {
		bytes[i] = b
	}
	return len(src), nil
}

func TestSelectMessages(t *testing.T) {
	inc := make(chan api.AnnotatedMessage)
	out := make(chan api.AnnotatedMessage)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	go selectMessages(ctx, os.Stdout, inc, out)

	inc <- api.AnnotatedMessage{}
	out <- api.AnnotatedMessage{}
}

func TestListenFromUser(t *testing.T) {
	messages := make(chan api.AnnotatedMessage)

	r := strings.NewReader("hak ben")

	go listenFromUser(r, messages, "topic")

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatal("test failed")
	case msg := <-messages:
		t.Logf("test passed: %v", msg)
		break
	}
}

func TestListenFromBroker(t *testing.T) {
	messages := make(chan api.AnnotatedMessage)

	reader := &mockReader{}
	go listenFromBroker(reader, messages)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatal("test failed")
	case msg := <-messages:
		t.Logf("test passed: %v", msg)
		break
	}
}

func TestSubscribe(t *testing.T) {
	if err := subscribe(os.Stdout, "testing", "default"); err != nil {
		t.Errorf("test failed: %v", err)
	}
}
