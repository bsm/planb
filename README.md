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

		if err := batch.Put([]byte(cmd.Args[0]), []byte(cmd.Args[1])); err != nil {
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

		val, err := batch.Get([]byte(cmd.Args[0]))
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
