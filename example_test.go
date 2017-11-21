package planb_test

import (
	"bytes"
	"fmt"

	"github.com/bsm/planb"
	"github.com/bsm/redeo/resp"
	"github.com/hashicorp/raft"
)

func ExampleServer() {
	// Open a store
	store := planb.NewInmemStore()

	// Init config
	conf := planb.NewConfig()
	conf.Sentinel.MasterName = "mymaster" // handle SENTINEL commands

	// Init server
	srv, err := planb.NewServer("10.0.0.1:7230", ".", store, raft.NewInmemStore(), raft.NewInmemStore(), conf)
	if err != nil {
		panic(err)
	}

	// Setup SET handler
	srv.HandleRW("SET", nil, planb.HandlerFunc(func(cmd *planb.Command) interface{} {
		if len(cmd.Args) != 2 {
			return fmt.Errorf("wrong number of arguments for '%q'", cmd.Name)
		}

		if err := store.Put(cmd.Args[0], cmd.Args[1]); err != nil {
			return err
		}
		return "OK"
	}))

	// Setup GET handler
	srv.HandleRO("GET", nil, planb.HandlerFunc(func(cmd *planb.Command) interface{} {
		if len(cmd.Args) != 1 {
			return fmt.Errorf("wrong number of arguments for '%q'", cmd.Name)
		}

		val, err := store.Get(cmd.Args[0])
		if err != nil {
			return err
		}
		return val
	}))

	// Start serving
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}

func ExampleCustomResponseFunc() {
	data := struct {
		Num int
		OK  bool
	}{
		Num: 2,
		OK:  true,
	}

	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)
	planb.CustomResponseFunc(func(w resp.ResponseWriter) {
		w.AppendArrayLen(4)
		w.AppendBulkString("num")
		w.AppendInt(int64(data.Num))
		w.AppendBulkString("ok")
		if data.OK {
			w.AppendInt(1)
		} else {
			w.AppendInt(0)
		}
	})(w)

	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "*4\r\n$3\r\nnum\r\n:2\r\n$2\r\nok\r\n:1\r\n"
}
