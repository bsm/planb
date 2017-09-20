package planb

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bsm/redeo"
	"github.com/bsm/redeo/info"
	"github.com/bsm/redeo/resp"
	"github.com/bsm/redeoraft"
	"github.com/hashicorp/raft"
)

// Server implements a peer
type Server struct {
	addr raft.ServerAddress
	rsrv *redeo.Server
	ctrl *raft.Raft
	kvs  KVStore

	handlers    map[string]Handler
	closeOnExit []func() error
}

// NewServer initializes a new server instance. Each server
// must advertise an address and use a local dir location
// and a key-value store for persistence.
// It also accepts a log and a stable store.
func NewServer(advertise raft.ServerAddress, dir string, kvs KVStore, logs raft.LogStore, stable raft.StableStore, conf *raft.Config) (*Server, error) {
	// ensure dir is created
	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, err
	}

	// create/normalise config
	if conf == nil {
		conf = raft.DefaultConfig()
	}
	if err := normNodeID(conf, filepath.Join(dir, "node-id")); err != nil {
		return nil, err
	}

	// init server
	s := &Server{
		addr:     advertise,
		rsrv:     redeo.NewServer(nil),
		kvs:      kvs,
		handlers: make(map[string]Handler),
	}

	// init RAFT stable snapshots
	snaps, err := raft.NewFileSnapshotStoreWithLogger(filepath.Join(dir, "snap"), 2, conf.Logger)
	if err != nil {
		_ = s.Close()
		return nil, err
	}

	// init RAFT transport
	trans := redeoraft.NewTransport(s.rsrv, advertise, &redeoraft.Options{
		Timeout: time.Second,
	})
	s.closeOnExit = append(s.closeOnExit, trans.Close)

	// init RAFT controller
	ctrl, err := raft.NewRaft(conf, &fsmWrapper{Server: s}, logs, stable, snaps, trans)
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	s.ctrl = ctrl
	s.closeOnExit = append(s.closeOnExit, func() error { return ctrl.Shutdown().Error() })

	// expose more info
	sinf := s.rsrv.Info().Section("Server")
	sinf.Register("node_id", info.StringValue(conf.LocalID))
	sinf.Register("tcp_addr", info.StringValue(advertise))

	// install default commands
	s.rsrv.Handle("ping", redeo.Ping())
	s.rsrv.Handle("info", redeo.Info(s.rsrv))
	s.rsrv.Handle("raftleader", redeoraft.Leader(ctrl))
	s.rsrv.Handle("raftstats", redeoraft.Stats(ctrl))
	s.rsrv.Handle("raftstate", redeoraft.State(ctrl))
	s.rsrv.Handle("raftpeers", redeoraft.Peers(ctrl))
	s.rsrv.HandleFunc("raftbootstrap", s.bootstrap)

	return s, nil
}

// ListenAndServe starts listening and serving
// on the advertised address.
func (s *Server) ListenAndServe() error {
	lis, err := net.Listen("tcp", string(s.addr))
	if err != nil {
		return err
	}
	defer lis.Close()

	return s.Serve(lis)
}

// Serve starts serving in the given listener
func (s *Server) Serve(lis net.Listener) error { return s.rsrv.Serve(lis) }

// KV exposes the internal KVStore
func (s *Server) KV() KVStore { return s.kvs }

// EnableSentinel enables sentinel support using the given master name,
// defaults to "mymaster".
func (s *Server) EnableSentinel(name string) {
	s.rsrv.Handle("sentinel", redeoraft.Sentinel(name, s.ctrl))
}

// HandleRO handles read-only commands
func (s *Server) HandleRO(name string, h Handler) {
	s.rsrv.Handle(name, readOnlyHandler{h: h})
}

// HandleRW handles read/write commands. These can only be applied to the master node
// and are replicated to slaves. An optional timeout can be specified to
// indicate the maximum the duration the server is willing to wait for the application
// of the command. Minimum: 1s, default: 10s.
func (s *Server) HandleRW(name string, timeout time.Duration, h Handler) {
	if timeout < time.Second {
		timeout = 10 * time.Second
	}
	s.handlers[strings.ToLower(name)] = h
	s.rsrv.Handle(name, replicatingHandler{s: s, timeout: timeout})
}

// Close closes the server
func (s *Server) Close() error {
	var err error
	for _, fn := range s.closeOnExit {
		if e := fn(); e != nil {
			err = e
		}
	}
	s.closeOnExit = nil
	return err
}

func (s *Server) bootstrap(w resp.ResponseWriter, c *resp.Command) {
	if c.ArgN() < 1 {
		w.AppendError(redeo.WrongNumberOfArgs(c.Name))
		return
	}

	servers := make([]raft.Server, c.ArgN())
	for i, arg := range c.Args() {
		addr := arg.String()
		conf, err := retrieveServerConfig(addr)
		if err != nil {
			w.AppendErrorf("ERR unable to retrieve info from %s: %s", addr, err.Error())
			return
		}
		servers[i] = *conf
	}

	if err := s.ctrl.BootstrapCluster(raft.Configuration{Servers: servers}).Error(); err != nil {
		w.AppendErrorf("ERR unable to bootstrap cluster: %s", err.Error())
		return
	}

	w.AppendOK()
}
