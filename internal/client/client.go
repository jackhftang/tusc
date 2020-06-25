package client

import (
  "fmt"
  "github.com/docopt/docopt-go"
  "github.com/eventials/go-tus"
  "github.com/eventials/go-tus/leveldbstore"
  "github.com/jackhftang/tusc/internal/util"
  "net/http"
  "os"
  "strings"
)

const clientUsage = `tusc client

Usage:
  tusc (client|c) <url> <file> [-H <header>]... [options]
  tusc (client|c) --help

Options:
  -r --resumable            Save meta data for resumable uploads [default: false]
  --store PATH              Path to save meta data for resume [default: ./.tusc]
  --chunk-size BYTE         Size of chunks of file [default: 2097152]
  --override-patch-method   Sending a POST request instead of PATCH [default: false]
`

type ClientConf struct {
  file   string
  url    string
  resume bool
}

func Client() {
  var err error

  var conf ClientConf
  arguments, _ := docopt.ParseDoc(clientUsage)
  conf.file, _ = arguments.String("<file>")
  conf.url, _ = arguments.String("<url>")
  conf.resume = util.GetBool(arguments, "--resumable")

  // open file
  f, err := os.Open(conf.file)
  if err != nil {
    util.ExitWithMessages("Cannot open file: " + conf.file)
  }
  defer f.Close()

  // create the tus client
  var store tus.Store
  if conf.resume {
    path := util.GetString(arguments, "--store")
    store, err = leveldbstore.NewLeveldbStore(path)
    if err != nil {
      util.ExitWithMessages("Cannot Open "+path, clientUsage)
    }
  }

  // create HTTP header
  header := make(http.Header)
  nHdr := arguments["-H"].(int)
  hdrs := util.GetSliceString(arguments, "<header>")
  for i := 0; i < nHdr; i += 1 {
    h := hdrs[i]
    ix := strings.Index(h, ":")
    if ix < 0 {
      continue
    }
    hdr := h[0:ix]
    val := h[ix+1:]
    fmt.Println(hdr)
    header.Set(hdr, val)
  }

  client, _ := tus.NewClient(conf.url, &tus.Config{
    ChunkSize:           util.GetInt64(arguments, "--chunk-size"),
    OverridePatchMethod: util.GetBool(arguments, "--override-patch-method"),
    Resume:              conf.resume,
    Store:               store,
    Header:              header,
    HttpClient:          nil,
  })

  // create an upload from a file.
  var upload *tus.Upload
  if upload, err = tus.NewUploadFromFile(f); err != nil {
    util.ExitWithMessages("Cannot create upload from file: " + f.Name())
  }

  // create the uploader.
  var uploader *tus.Uploader
  if conf.resume {
    uploader, err = client.CreateOrResumeUpload(upload)
  } else {
    uploader, err = client.CreateUpload(upload)
  }
  if err != nil {
    util.ExitWithMessages("Failed to upload", err.Error())
  }

  fmt.Println(uploader.Url())

  // start the uploading process.
  if err = uploader.Upload(); err != nil {
    util.ExitWithMessages("Upload incomplete", err.Error())
  }
}
