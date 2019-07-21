package main

import (
  "github.com/jackhftang/tusc/internal"
  "os"
)

const usage = `Usage:
  tusc (server|s) [options]
  tusc (client|c) <file> [<url>] [options]
  tusc --help`

func main() {
  if len(os.Args) < 2 {
    internal.ExitWithMessages("No command", usage)
  }
  switch cmd := os.Args[1]; cmd {
  case "server", "s":
    //internal.Server()
  case "client", "c":
    //internal.Client()
  default:
    internal.ExitWithMessages("Unknown command: "+cmd, usage)
  }
}
