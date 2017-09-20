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

func main() {{ "ExampleServer" | code }}
```
