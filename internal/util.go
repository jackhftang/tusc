package internal

import (
  "fmt"
  "github.com/docopt/docopt-go"
  "log"
  "os"
  "strconv"
)

var stdout = log.New(os.Stdout, "[tusd] ", log.Ldate|log.Ltime)
var stderr = log.New(os.Stderr, "[tusd] ", log.Ldate|log.Ltime)

//func logEv(logOutput *log.Logger, eventName string, details ...string) {
//  tusd.LogEvent(logOutput, eventName, details...)
//}

func ExitWithMessages(msg ...interface{}) {
  for _, m := range msg {
    fmt.Fprintln(os.Stderr, m)
  }
  //fmt.Fprintln(os.Stderr, msg...)
  os.Exit(1)
}

func getInt64(arg docopt.Opts, key string) int64 {
  s, _ := arg.String(key)
  i, _ := strconv.ParseInt(s, 10, 64)
  return i
}

func getBool(arg docopt.Opts, key string) bool {
  v, _ := arg[key]
  if b, ok := v.(bool); ok {
    return b
  }
  return false
}

func getString(arg docopt.Opts, key string) string {
  v, _ := arg[key]
  if s, ok := v.(string); ok {
    return s
  }
  return ""
}
