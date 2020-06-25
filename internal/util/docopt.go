package util

import (
  "github.com/docopt/docopt-go"
  "strconv"
)

func GetInt64(arg docopt.Opts, key string) int64 {
  s, _ := arg.String(key)
  i, _ := strconv.ParseInt(s, 10, 64)
  return i
}

func GetBool(arg docopt.Opts, key string) bool {
  v, _ := arg[key]
  if b, ok := v.(bool); ok {
    return b
  }
  return false
}

func GetString(arg docopt.Opts, key string) string {
  v, _ := arg[key]
  if s, ok := v.(string); ok {
    return s
  }
  return ""
}

func GetSliceString(arg docopt.Opts, key string) []string {
  v, _ := arg[key]
  if s, ok := v.([]string); ok {
    return s
  }
  return []string{}
}
