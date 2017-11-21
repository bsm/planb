package planb_test

import (
	"fmt"
	"io/ioutil"
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

	var serve = func(cb func(string, client.Conn)) func() {
		return func() {
			// init dir
			dir, err := ioutil.TempDir("", "planb-test")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(dir)

			// init listener
			lis, err := net.Listen("tcp", "127.0.0.1:")
			Expect(err).NotTo(HaveOccurred())
			addr := lis.Addr().String()
			defer lis.Close()

			// init config
			conf := planb.NewConfig()
			conf.Raft.LogOutput = ioutil.Discard

			// setup server
			store := planb.NewInmemStore()
			rfs := raft.NewInmemStore()
			srv, err := planb.NewServer(raft.ServerAddress(addr), dir, store, rfs, rfs, conf)
			Expect(err).NotTo(HaveOccurred())
			defer srv.Close()

			// handle commands
			srv.HandleRO("echo", nil, planb.HandlerFunc(func(cmd *planb.Command) interface{} {
				if len(cmd.Args) < 1 {
					return fmt.Errorf("wrong number of arguments for '%s'", cmd.Name)
				}
				return cmd.Args[0]
			}))
			srv.HandleRO("now", nil, planb.HandlerFunc(func(cmd *planb.Command) interface{} {
				return time.Now().Unix()
			}))
			srv.HandleRW("reset", nil, planb.HandlerFunc(func(cmd *planb.Command) interface{} {
				return true
			}))

			// start server
			go srv.Serve(lis)

			pool, err := client.New(&pool.Options{InitialSize: 1}, func() (net.Conn, error) {
				return net.Dial("tcp", addr)
			})
			Expect(err).NotTo(HaveOccurred())
			defer pool.Close()

			conn, err := pool.Get()
			Expect(err).NotTo(HaveOccurred())
			cb(dir, conn)
		}
	}

	It("should create dir structure", serve(func(dir string, cn client.Conn) {
		Expect(filepath.Glob(filepath.Join(dir, "*"))).To(ConsistOf(
			dir+"/node-id",
			dir+"/snap",
		))

		data, err := ioutil.ReadFile(dir + "/node-id")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(HaveLen(36))
	}))

	It("should handle read-only commands", serve(func(dir string, cn client.Conn) {
		cn.WriteCmdString("ECHO", "HeLLo")
		Expect(cn.Flush()).To(Succeed())
		Expect(cn.ReadBulkString()).To(Equal("HeLLo"))

		cn.WriteCmdString("NOW")
		Expect(cn.Flush()).To(Succeed())
		Expect(cn.ReadInt()).To(BeNumerically("~", time.Now().Unix(), 2))
	}))

	It("should fail on read/write commands if not leader", serve(func(dir string, cn client.Conn) {
		cn.WriteCmdString("RESET")
		Expect(cn.Flush()).To(Succeed())
		Expect(cn.ReadError()).To(Equal("READONLY node is not the leader"))
	}))

})
