package internal

import (
  "fmt"
  "github.com/docopt/docopt-go"
  "github.com/eventials/go-tus"
  "github.com/eventials/go-tus/leveldbstore"
  "github.com/jackhftang/tusc/internal/util"
  "net/http"
  "os"
)

const clientUsage = `tusc client

Usage:
  tusc (client|c) <url> <file> [options]
  tusc (client|c) --help

Options:
  -r --resumable            Save meta data for resumable uploads  
  --store PATH              Path to save meta data for resume [default: ./.tusc]
  --chuck-size BYTE         Size of chucks of file [default: 2097152]
  --override-patch-method   Sending a POST request instead of PATCH [default: false]
`

func Client() {
  var err error
  arguments, _ := docopt.ParseDoc(clientUsage)

  file, _ := arguments.String("<file>")
  url, _ := arguments.String("<url>")
  resume := util.GetBool(arguments, "--resumable")

  // open file
  f, err := os.Open(file)
  if err != nil {
    util.ExitWithMessages("Cannot open file: " + file)
  }
  defer f.Close()

  // create the tus client
  var store tus.Store
  if resume {
    path := util.GetString(arguments, "--store")
    store, err = leveldbstore.NewLeveldbStore(path)
    if err != nil {
      util.ExitWithMessages("Cannot Open "+path, clientUsage)
    }
  }

  client, _ := tus.NewClient(url, &tus.Config{
    ChunkSize:           util.GetInt64(arguments, "--chuck-size"),
    OverridePatchMethod: util.GetBool(arguments, "--override-patch-method"),
    Resume:              resume,
    Store:               store,
    Header:              make(http.Header),
    HttpClient:          nil,
  })

  // create an upload from a file.
  var upload *tus.Upload
  if upload, err = tus.NewUploadFromFile(f); err != nil {
    util.ExitWithMessages("Cannot create upload from file: " + f.Name())
  }

  // create the uploader.
  var uploader *tus.Uploader
  if resume {
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
    util.ExitWithMessages("Upload incomplete")
  }
}
