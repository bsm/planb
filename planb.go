package planb

import (
	"io"
	"time"

	"github.com/hashicorp/raft"
)

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
