package internal

import (
  "fmt"
  "github.com/bmizerany/pat"
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
  -u --url URL                    Url of HTTP server [default: http://localhost:1080]
  -b --bind ADDR                  Address to bind HTTP server to [default: 0.0.0.0]
  -p --port PORT                  Port to bind HTTP server to [default: 1080]
  -d --dir PATH                   Directory to store uploads in [default: ./data]
  --listing-endpoint PATH         Http path for flies listing [default: /]
  --files-endpoint PATH           Http path for files [default: /files/]
  --unix-sock PATH                If set will listen to a UNIX socket at this location instead of a TCP socket
  --max-size SIZE                 Maximum size of a single upload in bytes [default: 0]
  --store-size BYTE               Size of space allowed for storage [default: 0]
  --timeout TIMEOUT               Read timeout for connections in milliseconds.  A zero value means that reads will not timeout [default: 30*1000]
  --behind-proxy                  Respect X-Forwarded-* and similar headers which may be set by proxies [default: false]
`

type ServerConf struct {
  Url             string `docopt:"--url"`
  BindAddr        string `docopt:"--bind"`
  Port            string `docopt:"--port"`
  HttpSock        string `docopt:"--unix-sock"`
  MaxSize         int64  `docopt:"--max-size"`
  UploadDir       string `docopt:"--dir"`
  StoreSize       int64  `docopt:"--store-size"`
  ListingEndpoint string `docopt:"--listing-endpoint"`
  FilesEndpoint   string `docopt:"--files-endpoint"`
  Timeout         int64  `docopt:"--timeout"`
  IsBehindProxy   bool   `docopt:"--behind-proxy"`
}

var stdout = log.New(os.Stdout, "[tusd] ", log.Ldate|log.Ltime)
var stderr = log.New(os.Stderr, "[tusd] ", log.Ldate|log.Ltime)

func logEv(logOutput *log.Logger, eventName string, details ...string) {
  tusd.LogEvent(logOutput, eventName, details...)
}

func Server() {
  var conf ServerConf
  arguments, _ := docopt.ParseDoc(serverUsage)
  //arguments.Bind(&conf) // todo: bug
  conf.Url, _ = arguments.String("--url")
  conf.BindAddr, _ = arguments.String("--bind")
  conf.Port, _ = arguments.String("--port")
  conf.HttpSock, _ = arguments.String("--unix-sock")
  conf.MaxSize = util.GetInt64(arguments, "--max-size")
  conf.UploadDir, _ = arguments.String("--dir")
  conf.StoreSize = util.GetInt64(arguments, "--store-size")
  conf.ListingEndpoint, _ = arguments.String("--listing-endpoint")
  conf.FilesEndpoint, _ = arguments.String("--files-endpoint")
  conf.Timeout = util.GetInt64(arguments, "--timeout")
  conf.IsBehindProxy, _ = arguments.Bool("--behind-proxy")
  fmt.Println(conf)

  storeCompoesr := tusd.NewStoreComposer()

  stdout.Printf("Using '%s' as directory storage.\n", conf.UploadDir)
  if err := os.MkdirAll(conf.UploadDir, os.FileMode(0774)); err != nil {
    stderr.Fatalf("Unable to ensure directory exists: %s", err)
  }
  store := filestore.New(conf.UploadDir)
  store.UseIn(storeCompoesr)

  if conf.StoreSize > 0 {
    limitedstore.New(conf.StoreSize, storeCompoesr.Core, storeCompoesr.Terminater).UseIn(storeCompoesr)
    stdout.Printf("Using %.2fMB as storage size.\n", float64(conf.StoreSize)/1024/1024)

    // We need to ensure that a single upload can fit into the storage size
    if conf.MaxSize > conf.StoreSize || conf.MaxSize == 0 {
      conf.MaxSize = conf.StoreSize
    }
  }

  stdout.Printf("Using %.2fMB as maximum size.\n", float64(conf.MaxSize)/1024/1024)

  // Address
  address := ""
  if conf.HttpSock != "" {
    address = conf.HttpSock
    stdout.Printf("Using %s as socket to listen.\n", address)
  } else {
    address = conf.BindAddr + ":" + conf.Port
    stdout.Printf("Using %s as address to listen.\n", address)
  }

  // Base path
  stdout.Printf("Using %s as the base path.\n", conf.FilesEndpoint)

  // show capabilities
  stdout.Printf(storeCompoesr.Capabilities())

  // tus handler
  handler, err := tusd.NewHandler(tusd.Config{
    MaxSize:                 conf.MaxSize,
    BasePath:                conf.FilesEndpoint,
    RespectForwardedHeaders: conf.IsBehindProxy,
    StoreComposer:           storeCompoesr,
    NotifyCompleteUploads:   false,
    NotifyTerminatedUploads: false,
    NotifyUploadProgress:    false,
    NotifyCreatedUploads:    false,
  })
  if err != nil {
    stderr.Fatalf("Unable to create handler: %s", err)
  }

  if conf.ListingEndpoint != conf.FilesEndpoint {
    mux := pat.New()
    mux.Get("/", listingHandler(conf, store))
    http.Handle(conf.ListingEndpoint, mux)
  }
  http.Handle(conf.FilesEndpoint, http.StripPrefix(conf.FilesEndpoint, handler))


  var listener net.Listener
  timeoutDuration := time.Duration(conf.Timeout) * time.Millisecond

  if conf.HttpSock != "" {
    if listener, err = util.NewUnixListener(address, timeoutDuration, timeoutDuration); err != nil {
      stderr.Fatalf("Unable to create listener: %s", err)
    }
  } else {
    if listener, err = util.NewListener(address, timeoutDuration, timeoutDuration); err != nil {
      stderr.Fatalf("Unable to create listener: %s", err)
    }
    stdout.Printf("You can now upload files to: http://%s%s", address, conf.FilesEndpoint)
  }

  if err = http.Serve(listener, nil); err != nil {
    stderr.Fatalf("Unable to serve: %s", err)
  }
}

func listingHandler(conf ServerConf, store filestore.FileStore) http.HandlerFunc {
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
{{ range .Infos }}<li><a href="{{ $.Conf.Url }}{{ $.Conf.FilesEndpoint }}{{ .ID }}">{{ index .MetaData "filename" }}</a></li>{{ end }}
  </ul>
  </body>
</html>{{end}}`)
  if err != nil {
    stderr.Fatalf("Unable to parse template: %s", err)
  }

  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    var err error
    var fileInfos []os.FileInfo

    // todo: here read once, calling GetInfo read another once
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
    data := struct {
      Infos []tusd.FileInfo
      Conf  ServerConf
    }{infos, conf,}
    if err = t.ExecuteTemplate(w, "listing", data); err != nil {
      //stderr.Fatalf("Unable to render template: %s", err)
      http.Error(w, "", 500)
      return
    }
  })
}
