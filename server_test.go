package planb_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/bsm/planb"
	"github.com/bsm/pool"
	"github.com/bsm/redeo/client"
	"github.com/hashicorp/raft"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	var subject *planb.Server
	var dir string

	const addr = "127.0.0.1:31313"

	var listen = func(cb func(cn client.Conn)) {
		lis, err := net.Listen("tcp", addr)
		Expect(err).NotTo(HaveOccurred())
		defer lis.Close()

		go subject.Serve(lis)

		pool, err := client.New(&pool.Options{InitialSize: 1}, func() (net.Conn, error) {
			return net.Dial("tcp", addr)
		})
		Expect(err).NotTo(HaveOccurred())
		defer pool.Close()

		conn, err := pool.Get()
		Expect(err).NotTo(HaveOccurred())
		defer pool.Put(conn)

		cb(conn)
	}

	BeforeEach(func() {
		var err error

		dir, err = ioutil.TempDir("", "planb")
		Expect(err).NotTo(HaveOccurred())

		conf := raft.DefaultConfig()
		conf.Logger = log.New(ioutil.Discard, "", 0)

		kvs := planb.NewInmemStore()
		rfs := raft.NewInmemStore()
		subject, err = planb.NewServer(addr, dir, kvs, rfs, rfs, conf)
		Expect(err).NotTo(HaveOccurred())

		subject.HandleRO("echo", planb.HandlerFunc(func(cmd *planb.Command) interface{} {
			if len(cmd.Args) < 1 {
				return fmt.Errorf("wrong number of arguments for '%s'", cmd.Name)
			}
			return cmd.Args[0]
		}))

		subject.HandleRW("now", 0, planb.HandlerFunc(func(cmd *planb.Command) interface{} {
			if len(cmd.Args) != 0 {
				return fmt.Errorf("wrong number of arguments for '%s'", cmd.Name)
			}
			return time.Now().Unix()
		}))
	})

	AfterEach(func() {
		Expect(subject.Close()).To(Succeed())
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	It("should create dir structure", func() {
		Expect(filepath.Glob(filepath.Join(dir, "*"))).To(ConsistOf(
			dir+"/node-id",
			dir+"/snap",
		))

		data, err := ioutil.ReadFile(dir + "/node-id")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(HaveLen(36))
	})

	It("should handle read-only commands", func() {
		listen(func(cn client.Conn) {
			cn.WriteCmdString("ECHO", "HeLLo")
			Expect(cn.Flush()).To(Succeed())
			Expect(cn.ReadBulkString()).To(Equal("HeLLo"))
		})
	})

	It("should fail on read/write commands", func() {
		listen(func(cn client.Conn) {
			cn.WriteCmdString("NOW")
			Expect(cn.Flush()).To(Succeed())
			Expect(cn.ReadError()).To(Equal("READONLY node is not the leader"))
		})
	})

})
