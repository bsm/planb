package planb_test

import (
	"bytes"
	"fmt"

	"github.com/bsm/planb"
	"github.com/bsm/redeo/resp"
	"github.com/hashicorp/raft"
)

func ExampleServer() {
	// Init server
	srv, err := planb.NewServer("10.0.0.1:7230", ".", planb.NewInmemStore(), raft.NewInmemStore(), raft.NewInmemStore(), nil)
	if err != nil {
		panic(err)
	}

	// Handle SENTINEL commands
	srv.EnableSentinel("mymaster")

	// Setup SET handler
	srv.HandleRW("SET", 0, planb.HandlerFunc(func(cmd *planb.Command) interface{} {
		if len(cmd.Args) != 2 {
			return fmt.Errorf("wrong number of arguments for '%q'", cmd.Name)
		}

		batch, err := srv.KV().Begin(true)
		if err != nil {
			return err
		}
		defer batch.Rollback()

		if err := batch.Put(cmd.Args[0], cmd.Args[1]); err != nil {
			return err
		}
		if err := batch.Commit(); err != nil {
			return err
		}

		return "OK"
	}))

	// Setup GET handler
	srv.HandleRO("GET", planb.HandlerFunc(func(cmd *planb.Command) interface{} {
		if len(cmd.Args) != 1 {
			return fmt.Errorf("wrong number of arguments for '%q'", cmd.Name)
		}

		batch, err := srv.KV().Begin(false)
		if err != nil {
			return err
		}
		defer batch.Rollback()

		val, err := batch.Get(cmd.Args[0])
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
