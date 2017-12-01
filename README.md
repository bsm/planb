# Plan B

[![GoDoc](https://godoc.org/github.com/bsm/planb?status.svg)](https://godoc.org/github.com/bsm/planb)
[![Build Status](https://travis-ci.org/bsm/planb.png?branch=master)](https://travis-ci.org/bsm/planb)
[![Go Report Card](https://goreportcard.com/badge/github.com/bsm/planb)](https://goreportcard.com/report/github.com/bsm/planb)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Plan B is a toolkit for building distributed, low-latency services that speak [RESP](https://redis.io/topics/protocol)
(REdis Serialization Protocol). Under the hood, it is wrapping [Redeo](https://github.com/bsm/redeo) and
[Raft](https://github.com/hashicorp/raft) to create a concise interface for custom commands.

## Examples

A simple server example:

```go
package main

import (
  "fmt"

  "github.com/bsm/planb"
  "github.com/hashicorp/raft"

)

func main() {
	// Open a store
	store := planb.NewInmemStore()

	// Init config
	conf := planb.NewConfig()
	conf.Sentinel.MasterName = "mymaster"	// handle SENTINEL commands

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
```
