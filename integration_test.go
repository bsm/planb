package planb_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/bsm/planb"
	"github.com/bsm/redeo/client"
	"github.com/bsm/redeo/resp"
	"github.com/hashicorp/raft"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server (integration)", func() {
	var nodes testNodes
	var leader, follower *testNode

	var skipOnShort = func(cb func()) func() {
		return func() {
			if !testing.Short() {
				cb()
			}
		}
	}

	BeforeEach(skipOnShort(func() {
		var err error

		nodes = make(testNodes, 3)
		for i := 0; i < 3; i++ {
			nodes[i], err = newTestNode()
			Expect(err).NotTo(HaveOccurred())
		}
		for _, n := range nodes {
			Expect(n.Cmd("raftbootstrap", nodes[0].Addr(), nodes[1].Addr(), nodes[2].Addr())).To(Equal("OK"))
		}

		Eventually(func() (string, error) {
			return nodes[0].Cmd("raftleader")
		}, "10s").ShouldNot(BeEmpty())

		leader, err = nodes.Find("leader")
		Expect(err).NotTo(HaveOccurred())

		follower, err = nodes.Find("follower")
		Expect(err).NotTo(HaveOccurred())
	}))

	AfterEach(skipOnShort(func() {
		for _, n := range nodes {
			n.Close()
		}
	}))

	It("should boot and elect leader", skipOnShort(func() {
		Expect(nodes[0].Cmd("raftleader")).To(Equal(leader.Addr()))
	}))

	It("should accept writes on leader and replicate to followers", skipOnShort(func() {
		Expect(follower.Cmd("SET", "key", "v1")).To(Equal("READONLY node is not the leader"))

		Expect(leader.Cmd("SET", "key", "v1")).To(Equal("OK"))
		Expect(leader.Cmd("GET", "key")).To(Equal("v1"))
		Eventually(func() (string, error) { return follower.Cmd("GET", "key") }).Should(Equal("v1"))

		Expect(leader.Cmd("SET", "key", "v2")).To(Equal("OK"))
		Eventually(func() (string, error) { return follower.Cmd("GET", "key") }).Should(Equal("v2"))
	}))

})

// --------------------------------------------------------------------

type testNodes []*testNode

func (nn testNodes) Find(s string) (*testNode, error) {
	for _, n := range nn {
		x, err := n.Cmd("raftstate")
		if err != nil {
			return nil, err
		}

		if s == x {
			return n, nil
		}
	}
	return nil, fmt.Errorf("unable to find a %s node", s)
}

// --------------------------------------------------------------------

type testNode struct {
	lis net.Listener
	dir string
	srv *planb.Server
	cln *client.Pool
	kvs *planb.InmemStore
}

func newTestNode() (*testNode, error) {
	var err error

	node := &testNode{kvs: planb.NewInmemStore()}
	node.dir, err = ioutil.TempDir("", "planb-test-node")
	if err != nil {
		node.Close()
		return nil, err
	}

	node.lis, err = net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		node.Close()
		return nil, err
	}

	conf := planb.NewConfig()
	conf.Raft.LogOutput = ioutil.Discard

	node.srv, err = planb.NewServer(raft.ServerAddress(node.Addr()), node.dir, node.kvs, raft.NewInmemStore(), raft.NewInmemStore(), conf)
	if err != nil {
		node.Close()
		return nil, err
	}

	node.cln, err = client.New(nil, func() (net.Conn, error) { return net.Dial("tcp", node.Addr()) })
	if err != nil {
		node.Close()
		return nil, err
	}

	node.srv.HandleRW("set", nil, planb.HandlerFunc(node.handleSet))
	node.srv.HandleRO("get", nil, planb.HandlerFunc(node.handleGet))

	go node.srv.Serve(node.lis)
	return node, err
}

func (n *testNode) Addr() string { return n.lis.Addr().String() }

func (n *testNode) Cmd(name string, args ...string) (string, error) {
	cn, err := n.cln.Get()
	if err != nil {
		return "", err
	}
	defer n.cln.Put(cn)

	cn.WriteCmdString(name, args...)
	if err := cn.Flush(); err != nil {
		cn.MarkFailed()
		return "", err
	}

	t, err := cn.PeekType()
	if err != nil {
		return "", err
	}

	switch t {
	case resp.TypeInline:
		return cn.ReadInlineString()
	case resp.TypeError:
		return cn.ReadError()
	default:
		return cn.ReadBulkString()
	}
}

func (n *testNode) Close() {
	if n.cln != nil {
		_ = n.cln.Close()
		n.cln = nil
	}
	if n.srv != nil {
		_ = n.srv.Close()
		n.srv = nil
	}
	if n.lis != nil {
		_ = n.lis.Close()
		n.lis = nil
	}
	if n.dir != "" {
		_ = os.RemoveAll(n.dir)
		n.dir = ""
	}
}

func (n *testNode) handleSet(cmd *planb.Command) interface{} {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("wrong number of arguments for '%q'", cmd.Name)
	}
	if err := n.kvs.Put(cmd.Args[0], cmd.Args[1]); err != nil {
		return err
	}
	return "OK"
}

func (n *testNode) handleGet(cmd *planb.Command) interface{} {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("wrong number of arguments for '%q'", cmd.Name)
	}

	val, err := n.kvs.Get(cmd.Args[0])
	if err != nil {
		return err
	}
	return val
}
