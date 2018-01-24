package planb

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bsm/redeo/resp"
	"github.com/hashicorp/raft"
)

var bufPool = sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}

// --------------------------------------------------------------------

type fsmWrapper struct{ *Server }

func (f *fsmWrapper) Apply(log *raft.Log) interface{} {
	var cmd resp.Command
	if err := gob.NewDecoder(bytes.NewReader(log.Data)).Decode(&cmd); err != nil {
		return err
	}

	h, ok := f.handlers[strings.ToLower(cmd.Name)]
	if !ok {
		return fmt.Errorf("unknown command '%s'", cmd.Name)
	}

	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()

	w := resp.NewResponseWriter(b)
	h.ServeRedeo(w, &cmd)

	if err := w.Flush(); err != nil {
		bufPool.Put(b)
		return err
	}
	return b
}

func (f *fsmWrapper) Restore(rc io.ReadCloser) error      { return f.store.Restore(rc) }
func (f *fsmWrapper) Snapshot() (raft.FSMSnapshot, error) { return &fsmSnapshot{Store: f.store}, nil }

type fsmSnapshot struct{ Store }

func (s *fsmSnapshot) Release() {}
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	if err := s.Snapshot(sink); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}

// --------------------------------------------------------------------

type replicatingHandler struct {
	s *Server
	o *HandlerOpts
}

func (h replicatingHandler) ServeRedeo(w resp.ResponseWriter, c *resp.Command) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := gob.NewEncoder(buf).Encode(c); err != nil {
		w.AppendError("ERR " + err.Error())
		return
	}

	future := h.s.ctrl.Apply(buf.Bytes(), h.o.getTimeout())
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

	switch res := future.Response().(type) {
	case *bytes.Buffer:
		if _, err := res.WriteTo(w); err != nil {
			w.AppendError("ERR " + err.Error())
		}
		bufPool.Put(res)
	case error:
		w.AppendError("ERR " + err.Error())
	default:
		w.AppendNil()
	}
}
