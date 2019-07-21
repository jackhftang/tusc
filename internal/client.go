package internal

import (
  "fmt"
  "github.com/docopt/docopt-go"
  "github.com/eventials/go-tus"
  "github.com/eventials/go-tus/leveldbstore"
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
  arguments, _ := docopt.ParseDoc(clientUsage)

  file, _ := arguments.String("<file>")
  url, _ := arguments.String("<url>")
  resume := getBool(arguments, "--resumable")

  f, err := os.Open(file)
  if err != nil {
    ExitWithMessages("Cannot open file: " + file)
  }
  defer f.Close()

  // create the tus client
  var store tus.Store
  if resume {
    path := getString(arguments, "--store")
    store, err = leveldbstore.NewLeveldbStore(path)
    if err != nil {
      ExitWithMessages("Cannot Open "+path, clientUsage)
    }
  }

  client, _ := tus.NewClient(url, &tus.Config{
    ChunkSize:           getInt64(arguments, "--chuck-size"),
    OverridePatchMethod: getBool(arguments, "--override-patch-method"),
    Resume:              resume,
    Store:               store,
    Header:              make(http.Header),
    HttpClient:          nil,
  })

  // create an upload from a file.
  upload, _ := tus.NewUploadFromFile(f)

  // create the uploader.
  var uploader *tus.Uploader
  if resume {
    uploader, err = client.CreateOrResumeUpload(upload)
  } else {
    uploader, err = client.CreateUpload(upload)
  }

  fmt.Println(uploader.Url())

  // start the uploading process.
  if err = uploader.Upload(); err != nil {
    ExitWithMessages("Upload incomplete")
  }
}
