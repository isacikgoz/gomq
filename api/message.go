package api

import "encoding/json"

// AnnotatedMessage consists of the bare message and headers
type AnnotatedMessage struct {
	Sender  string          `json:"sender"`  // sender name
	Command string          `json:"command"` // pub/sub
	Target  string          `json:"target"`  // defines the topic/or queue name
	Payload json.RawMessage `json:"Payload"` // the original message
}
