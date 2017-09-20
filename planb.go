package planb

import (
	"io"

	"github.com/bsm/redeo/resp"
)

// Command represents a command sent by the client
// to the server.
type Command struct {
	// Name is the command name
	Name string
	// Args are the command arguments
	Args []resp.CommandArgument
}

// Handler is a protocol for handling and responding incoming commands. The response
// returned by a handler must be one of the following types:
//
//   nil
//   error
//   string
//   []byte
//   bool (bools are returned as 0/1 by the server)
//   int/int8/int16/int32/int64
//   float32/float64
//   []string
//   [][]byte
//   []int
//   []int64
//   map[string]string
//
// Additonally interfaces implementing CustomResponse could be returned too.
type Handler interface {
	// ServeRequest responds to an incoming command
	// and generates a response.
	ServeRequest(cmd *Command) interface{}
}

// HandlerFunc allows to wrap simple Handlers in functions
type HandlerFunc func(cmd *Command) interface{}

// ServeRequest implements Handler
func (f HandlerFunc) ServeRequest(cmd *Command) interface{} { return f(cmd) }

// Store is an abstraction of an underlying
// store implementation. It must have snapshot
// and restore capabilities.
type Store interface {
	// Restore restores the store from a data stream.
	Restore(r io.Reader) error
	// Snapshot writes a snapshot to the provided writer.
	Snapshot(w io.Writer) error
}
