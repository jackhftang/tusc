package main

import (
  "github.com/jackhftang/tusc/internal"
  "github.com/jackhftang/tusc/internal/util"
  "os"
)

const usage = `Usage:
  tusc (server|s) [options]
  tusc (client|c) <file> [<url>] [options]
  tusc --help`

func main() {
  if len(os.Args) < 2 {
    util.ExitWithMessages("No command", usage)
  }
  switch cmd := os.Args[1]; cmd {
  case "server", "s":
    internal.Server()
  case "client", "c":
    internal.Client()
  default:
    util.ExitWithMessages("Unknown command: "+cmd, usage)
  }
}
