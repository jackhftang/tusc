package internal

import (
  "fmt"
  "github.com/docopt/docopt-go"
  "github.com/jackhftang/tusc/internal/util"
  "github.com/tus/tusd"
  "github.com/tus/tusd/filestore"
  "github.com/tus/tusd/limitedstore"
  "html/template"
  "io/ioutil"
  "log"
  "net"
  "net/http"
  "os"
  "sort"
  "time"
)

const serverUsage = `tusc server

Usage:
  tusc (server|s) [options] 
  tusc (server|s) --help

Options:
  -h --host HOST                  Host to bind HTTP server to [default: 0.0.0.0]
  -p --port PORT                  Port to bind HTTP server to [default: 1080]
  -d --dir PATH                   Directory to store uploads in [default: ./data]
  -b --base-path PATH             Basepath of the HTTP server [default: /files/]
  --unix-sock PATH                If set will listen to a UNIX socket at this location instead of a TCP socket
  --max-size SIZE                 Maximum size of a single upload in bytes [default: 0]
  --store-size BYTE               Size of space allowed for storage [default: 0]
  --timeout TIMEOUT               Read timeout for connections in milliseconds.  A zero value means that reads will not timeout [default: 30*1000]
  --behind-proxy                  Respect X-Forwarded-* and similar headers which may be set by proxies [default: false]
`

var stdout = log.New(os.Stdout, "[tusd] ", log.Ldate|log.Ltime)
var stderr = log.New(os.Stderr, "[tusd] ", log.Ldate|log.Ltime)

func logEv(logOutput *log.Logger, eventName string, details ...string) {
  tusd.LogEvent(logOutput, eventName, details...)
}

func Server() {
  arguments, _ := docopt.ParseDoc(serverUsage)
  var httpHost, _ = arguments.String("--host")
  var httpPort, _ = arguments.String("--port")
  var httpSock, _ = arguments.String("--unix-sock")
  var maxSize = util.GetInt64(arguments, "--max-size")
  var uploadDir, _ = arguments.String("--dir")
  var storeSize = util.GetInt64(arguments, "--store-size")
  var listingEndpoint = "/"
  var uploadEndpoint, _ = arguments.String("--base-path")
  var timeout = util.GetInt64(arguments, "--timeout")
  var isBehindProxy, _ = arguments.Bool("--behind-proxy")

  storeCompoesr := tusd.NewStoreComposer()

  stdout.Printf("Using '%s' as directory storage.\n", uploadDir)
  if err := os.MkdirAll(uploadDir, os.FileMode(0774)); err != nil {
    stderr.Fatalf("Unable to ensure directory exists: %s", err)
  }
  store := filestore.New(uploadDir)
  store.UseIn(storeCompoesr)

  if storeSize > 0 {
    limitedstore.New(storeSize, storeCompoesr.Core, storeCompoesr.Terminater).UseIn(storeCompoesr)
    stdout.Printf("Using %.2fMB as storage size.\n", float64(storeSize)/1024/1024)

    // We need to ensure that a single upload can fit into the storage size
    if maxSize > storeSize || maxSize == 0 {
      maxSize = storeSize
    }
  }

  stdout.Printf("Using %.2fMB as maximum size.\n", float64(maxSize)/1024/1024)

  // Serve

  // Address
  address := ""
  if httpSock != "" {
    address = httpSock
    stdout.Printf("Using %s as socket to listen.\n", address)
  } else {
    address = httpHost + ":" + httpPort
    stdout.Printf("Using %s as address to listen.\n", address)
  }

  // Base path
  stdout.Printf("Using %s as the base path.\n", uploadEndpoint)

  // show capabilities
  stdout.Printf(storeCompoesr.Capabilities())

  // tus handler
  handler, err := tusd.NewHandler(tusd.Config{
    MaxSize:                 maxSize,
    BasePath:                uploadEndpoint,
    RespectForwardedHeaders: isBehindProxy,
    StoreComposer:           storeCompoesr,
    NotifyCompleteUploads:   false,
    NotifyTerminatedUploads: false,
    NotifyUploadProgress:    false,
    NotifyCreatedUploads:    false,
  })
  if err != nil {
    stderr.Fatalf("Unable to create handler: %s", err)
  }

  http.Handle(uploadEndpoint, http.StripPrefix(uploadEndpoint, handler))
  if listingEndpoint != uploadEndpoint {
    http.Handle(listingEndpoint, http.StripPrefix(listingEndpoint, homepage(store)))
  }

  var listener net.Listener
  timeoutDuration := time.Duration(timeout) * time.Millisecond

  if httpSock != "" {
    if listener, err = util.NewUnixListener(address, timeoutDuration, timeoutDuration); err != nil {
      stderr.Fatalf("Unable to create listener: %s", err)
    }
    stdout.Printf("You can now upload files to: http://%s%s", address, uploadEndpoint)
  } else {
    if listener, err = util.NewListener(address, timeoutDuration, timeoutDuration); err != nil {
      stderr.Fatalf("Unable to create listener: %s", err)
    }
  }

  if err = http.Serve(listener, nil); err != nil {
    stderr.Fatalf("Unable to serve: %s", err)
  }
}

func homepage(store filestore.FileStore) http.HandlerFunc {
  t, err := template.New("foo").Parse(`{{define "listing"}}<html><head><title>File Listing</title><style>
* {
  font-family: monospace;
  font-size: 18px;
  box-sizing: border-box;
}

a {
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}

a:visited {
  color: blue;
}

ul {
  list-style-type: none;
  margin: 0;
  padding: 0;
}

li {
  margin: 5px 10px;
  padding: 0;
}
</style></head><body><ul>
{{ range . }}<li><a href="http://127.0.0.1:1080/files/{{ .ID }}">{{ index .MetaData "filename" }}</a></li>{{ end }}
  </ul>
  </body>
</html>{{end}}`)
  if err != nil {
    stderr.Fatalf("Unable to parse template: %s", err)
  }

  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    var err error
    var fileInfos []os.FileInfo
    if fileInfos, err = ioutil.ReadDir(store.Path); err != nil {
      http.Error(w, "", 500)
      return
    }

    // collect file info
    var infos []tusd.FileInfo
    for _, f := range fileInfos {
      filename := f.Name()
      ext := ".info"
      lenOfID := len(filename) - len(ext)
      fmt.Println("filename", filename, filename[lenOfID:])

      // only care .bin file
      if lenOfID > 0 && filename[lenOfID:] == ext {
        if info, err := store.GetInfo(filename[:lenOfID]); err != nil {
          //stderr.Fatalf("Unable to get file info: %s", err)
          http.Error(w, "", 500)
          return
        } else {
          infos = append(infos, info)
          fmt.Println("info", info)
        }
      }
    }
    sort.Slice(infos, func(i, j int) bool {
      return infos[i].MetaData["filename"] < infos[j].MetaData["filename"]
    })
    if err = t.ExecuteTemplate(w, "listing", infos); err != nil {
      //stderr.Fatalf("Unable to render template: %s", err)
      http.Error(w, "", 500)
      return
    }
  })
}
