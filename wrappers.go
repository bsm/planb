package planb

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/bsm/redeo/resp"
	"github.com/hashicorp/raft"
)

var bufPool = sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}

// --------------------------------------------------------------------

type fsmWrapper struct{ *Server }

func (f *fsmWrapper) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := gob.NewDecoder(bytes.NewReader(log.Data)).Decode(&cmd); err != nil {
		return err
	}

	h, ok := f.handlers[strings.ToLower(cmd.Name)]
	if !ok {
		return fmt.Errorf("unknown command '%s'", cmd.Name)
	}

	return h.ServeRequest(&cmd)
}

func (f *fsmWrapper) Restore(rc io.ReadCloser) error      { return f.store.Restore(rc) }
func (f *fsmWrapper) Snapshot() (raft.FSMSnapshot, error) { return &fsmSnapshot{Store: f.store}, nil }

type fsmSnapshot struct{ Store }

func (s *fsmSnapshot) Release() {}
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	if err := s.Snapshot(sink); err != nil {
		return sink.Cancel()
	}
	return sink.Close()
}

// --------------------------------------------------------------------

type readOnlyHandler struct{ h Handler }

func (h readOnlyHandler) ServeRedeo(w resp.ResponseWriter, c *resp.Command) {
	v := h.h.ServeRequest(&Command{
		Name: c.Name,
		Args: c.Args(),
	})
	if v == nil {
		w.AppendNil()
	} else {
		respondWith(w, v)
	}
}

type replicatingHandler struct {
	s *Server

	timeout time.Duration
}

func (h replicatingHandler) ServeRedeo(w resp.ResponseWriter, c *resp.Command) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := gob.NewEncoder(buf).Encode(&Command{
		Name: c.Name,
		Args: c.Args(),
	}); err != nil {
		w.AppendError("ERR " + err.Error())
		return
	}

	future := h.s.ctrl.Apply(buf.Bytes(), h.timeout)
	err := future.Error()
	switch err {
	case raft.ErrNotLeader:
		w.AppendError("READONLY " + err.Error())
		return
	default:
		w.AppendError("ERR " + err.Error())
		return
	case nil:
	}

	if v := future.Response(); v == nil {
		w.AppendNil()
	} else {
		respondWith(w, v)
	}
}
