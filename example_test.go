package planb_test

import (
	"github.com/bsm/planb"
	"github.com/bsm/redeo"
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
	srv.HandleRW("SET", nil, redeo.WrapperFunc(func(cmd *resp.Command) interface{} {
		if len(cmd.Args) != 2 {
			return redeo.ErrWrongNumberOfArgs(cmd.Name)
		}

		if err := store.Put(cmd.Args[0], cmd.Args[1]); err != nil {
			return err
		}
		return "OK"
	}))

	// Setup GET handler
	srv.HandleRO("GET", nil, redeo.WrapperFunc(func(cmd *resp.Command) interface{} {
		if len(cmd.Args) != 1 {
			return redeo.ErrWrongNumberOfArgs(cmd.Name)
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
