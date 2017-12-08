package planb

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/bsm/redeo"
	"github.com/bsm/redeo/info"
	"github.com/bsm/redeo/resp"
	"github.com/bsm/redeoraft"
	"github.com/hashicorp/raft"
)

// Server implements a peer
type Server struct {
	addr  raft.ServerAddress
	rsrv  *redeo.Server
	ctrl  *raft.Raft
	store Store

	handlers    map[string]redeo.Handler
	closeOnExit []func() error
}

// NewServer initializes a new server instance. Each server
// must advertise an address and use a local dir location
// and a key-value store for persistence.
// It also accepts a log and a stable store.
func NewServer(advertise raft.ServerAddress, dir string, store Store, logs raft.LogStore, stable raft.StableStore, conf *Config) (*Server, error) {
	// ensure dir is created
	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, err
	}

	// create/normalise config
	if conf == nil {
		conf = NewConfig()
	}
	if err := conf.norm(filepath.Join(dir, "node-id")); err != nil {
		return nil, err
	}

	// init server
	s := &Server{
		addr:     advertise,
		rsrv:     redeo.NewServer(nil),
		store:    store,
		handlers: make(map[string]redeo.Handler),
	}

	// init RAFT stable snapshots
	snaps, err := raft.NewFileSnapshotStoreWithLogger(filepath.Join(dir, "snap"), 2, conf.Raft.Logger)
	if err != nil {
		_ = s.Close()
		return nil, err
	}

	// init RAFT transport
	trans := redeoraft.NewTransport(s.rsrv, advertise, conf.Transport)
	s.closeOnExit = append(s.closeOnExit, trans.Close)

	// init RAFT controller
	ctrl, err := raft.NewRaft(conf.Raft, &fsmWrapper{Server: s}, logs, stable, snaps, trans)
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	s.ctrl = ctrl
	s.closeOnExit = append(s.closeOnExit, func() error { return ctrl.Shutdown().Error() })

	// expose more info
	sinf := s.rsrv.Info().Section("Server")
	sinf.Register("node_id", info.StringValue(conf.Raft.LocalID))
	sinf.Register("tcp_addr", info.StringValue(advertise))

	// install default commands
	s.rsrv.Handle("ping", redeo.Ping())
	s.rsrv.Handle("info", redeo.Info(s.rsrv))
	s.rsrv.Handle("raft", redeo.SubCommands{
		"leader":    redeoraft.Leader(ctrl),
		"stats":     redeoraft.Stats(ctrl),
		"state":     redeoraft.State(ctrl),
		"peers":     redeoraft.Peers(ctrl),
		"add":       redeoraft.AddPeer(ctrl),
		"remove":    redeoraft.RemovePeer(ctrl),
		"bootstrap": redeo.HandlerFunc(s.bootstrap),
	})

	// Snables sentinel support if master name given.
	if name := conf.Sentinel.MasterName; name != "" {
		broker := redeo.NewPubSubBroker()
		s.rsrv.Handle("sentinel", redeoraft.Sentinel(name, ctrl, broker))
		s.rsrv.Handle("publish", broker.Publish())
		s.rsrv.Handle("subscribe", broker.Subscribe())
	}

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

// HandleRO handles readonly commands
func (s *Server) HandleRO(name string, opt *HandlerOpts, h redeo.Handler) {
	s.rsrv.Handle(name, h)
}

// HandleRW handles commands that may result in modifications. These can only be
// applied to the master node and are then replicated to slaves.
func (s *Server) HandleRW(name string, opt *HandlerOpts, h redeo.Handler) {
	s.handlers[strings.ToLower(name)] = h
	s.rsrv.Handle(name, replicatingHandler{s: s, o: opt})
}

// Raft exposes the underlying raft node controller
func (s *Server) Raft() RaftCtrl { return s.ctrl }

// Info server info
func (s *Server) Info() *redeo.ServerInfo { return s.rsrv.Info() }

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
	for i, arg := range c.Args {
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
