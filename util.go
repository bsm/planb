package planb

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/bsm/pool"
	"github.com/bsm/redeo/client"
	"github.com/bsm/redeo/resp"
	"github.com/google/uuid"
	"github.com/hashicorp/raft"
)

var errUnexpectedServerResponse = errors.New("unexpected response")

// "inspired" by https://github.com/hashicorp/consul
func normNodeID(conf *raft.Config, fname string) error {
	nodeID := string(conf.LocalID)

	// if we don't have a nodeID, try to read it from file
	if nodeID == "" {
		if _, err := os.Stat(fname); err == nil {
			binID, err := ioutil.ReadFile(fname)
			if err != nil {
				return err
			}
			nodeID = string(binID)
		}
	}

	// if we have a node ID, normalise and validate it.
	if nodeID != "" {
		nodeID = strings.ToLower(strings.TrimSpace(nodeID))
		if _, err := uuid.Parse(nodeID); err != nil {
			return err
		}
		conf.LocalID = raft.ServerID(nodeID)
		return nil
	}

	// otherwise, create a new one
	nodeID = uuid.New().String()
	if err := ioutil.WriteFile(fname, []byte(nodeID), 0600); err != nil {
		return err
	}
	conf.LocalID = raft.ServerID(nodeID)
	return nil
}

func retrieveServerConfig(addr string) (*raft.Server, error) {
	pool, err := client.New(&pool.Options{InitialSize: 1}, func() (net.Conn, error) {
		return net.Dial("tcp", addr)
	})
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	cn, err := pool.Get()
	if err != nil {
		return nil, err
	}
	defer pool.Put(cn)

	cn.WriteCmd("INFO")
	if err := cn.Flush(); err != nil {
		cn.MarkFailed()
		return nil, err
	}

	typ, err := cn.PeekType()
	if err := cn.Flush(); err != nil {
		cn.MarkFailed()
		return nil, err
	}

	switch typ {
	case resp.TypeBulk:
	default:
		return nil, errUnexpectedServerResponse
	}

	raw, err := cn.ReadBulk(nil)
	if err != nil {
		cn.MarkFailed()
		return nil, err
	}

	info := serverInfo(raw)
	nodeID, err := info.NodeID()
	if err != nil {
		return nil, err
	}

	address, err := info.Address()
	if err != nil {
		return nil, err
	}

	return &raft.Server{
		ID:      nodeID,
		Address: address,
	}, nil
}

// --------------------------------------------------------------------

type serverInfo []byte

func (i serverInfo) NodeID() (raft.ServerID, error) {
	nodeID, err := i.parse("node_id")
	if err != nil {
		return "", err
	}
	if _, err := uuid.Parse(nodeID); err != nil {
		return "", err
	}
	return raft.ServerID(nodeID), nil
}

func (i serverInfo) Address() (raft.ServerAddress, error) {
	address, err := i.parse("tcp_addr")
	if err != nil {
		return "", err
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		return "", err
	}
	return raft.ServerAddress(address), nil
}

func (i serverInfo) parse(s string) (string, error) {
	raw := []byte(i)
	pivot := []byte("\n" + s + ":")

	if pos := bytes.Index(raw, pivot); pos < 0 {
		return "", errUnexpectedServerResponse
	} else {
		raw = raw[pos+len(pivot):]
	}

	if pos := bytes.Index(raw, []byte{'\n'}); pos < 0 {
		return "", errUnexpectedServerResponse
	} else {
		raw = raw[:pos]
	}
	return string(raw), nil
}

// --------------------------------------------------------------------

const (
	fnvOffset32 uint32 = 2166136261
	fnvPrime32  uint32 = 16777619
)

func fnv32a(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}

	h := fnvOffset32
	for _, c := range b {
		h ^= uint32(c)
		h *= fnvPrime32
	}
	return h
}
