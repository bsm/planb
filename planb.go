package planb

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bsm/redeo/resp"
	"github.com/hashicorp/raft"
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

// SubCommands returns a handler that is parsing sub-commands
type SubCommands map[string]Handler

// ServeRequest implements Handler
func (s SubCommands) ServeRequest(cmd *Command) interface{} {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("wrong number of arguments for '%s'", cmd.Name)
	}

	firstArg := cmd.Args[0].String()
	if h, ok := s[strings.ToLower(firstArg)]; ok {
		return h.ServeRequest(&Command{
			Name: cmd.Name + " " + firstArg,
			Args: cmd.Args[1:],
		})
	}

	return fmt.Errorf("Unknown %s subcommand '%s'", strings.ToLower(cmd.Name), firstArg)
}

// --------------------------------------------------------------------

// HandlerOpts contain options for handler execution.
type HandlerOpts struct {
	// Timeout is an optional timeout for mutating commands. It indicates
	// the maximum duration the server is willing to wait for the application
	// of the command. Minimum: 1s, default: 10s.
	Timeout time.Duration
}

func (o *HandlerOpts) getTimeout() time.Duration {
	if o != nil && o.Timeout >= time.Second {
		return o.Timeout
	}
	return 10 * time.Second
}

// --------------------------------------------------------------------

// Store is an abstraction of an underlying
// store implementation. It must have snapshot
// and restore capabilities.
type Store interface {
	// Restore restores the store from a data stream.
	Restore(r io.Reader) error
	// Snapshot writes a snapshot to the provided writer.
	Snapshot(w io.Writer) error
}

// RaftCtrl is an interface to the underlying raft node controller
type RaftCtrl interface {
	// AppliedIndex returns the last index applied to the FSM.
	AppliedIndex() uint64
	// GetConfiguration returns the latest configuration and its associated index currently in use.
	GetConfiguration() raft.ConfigurationFuture
	// LastContact returns the time of last contact by a leader.
	LastContact() time.Time
	// LastIndex returns the last index in stable storage, either from the last log or from the last snapshot.
	LastIndex() uint64
	// Leader is used to return the current leader of the cluster. It may return empty string if there is no current leader or the leader is unknown.
	Leader() raft.ServerAddress
	// State is used to return the current raft state.
	State() raft.RaftState
	// Stats is used to return a map of various internal stats.
	Stats() map[string]string
	// String returns a string representation of this Raft node.
	String() string
}
