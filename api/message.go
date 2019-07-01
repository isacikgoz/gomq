package api

import "encoding/json"

// AnnotatedMessage consists of the bare message and headers
type AnnotatedMessage struct {
	Command string          `json:"command"` // pub/sub
	Target  string          `json:"target"`  // defines the topic/or queue name
	Payload json.RawMessage `json:"Payload"` // the original message
}

// BareMessage is the deafult data type to be passed
type BareMessage struct {
	Message string
}
