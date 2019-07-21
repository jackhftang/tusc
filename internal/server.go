package internal

import (
  "errors"
  "github.com/docopt/docopt-go"
  "github.com/tus/tusd"
  "github.com/tus/tusd/cmd/tusd/cli"
  "github.com/tus/tusd/filestore"
  "github.com/tus/tusd/limitedstore"
  "net"
  "net/http"
  "os"
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
  --s3-bucket BUCKET              Use AWS S3 with this bucket as storage backend requires the AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY and AWS_REGION environment variables to be set
  --s3-object-prefix              Prefix for S3 object names
  --s3-endpoint PATH              Endpoint to use S3 compatible implementations like minio requires s3-bucket to be pass
  --gcs-bucket BUCKET             Use Google Cloud Storage with this bucket as storage backend requires the GCS_SERVICE_ACCOUNT_FILE environment variable to be set
  --gcs-object-prefix PREFIX      Prefix for GCS object names can't contain underscore character
  --hooks-dir PATH                Directory to search for available hooks scripts
  --hooks-http PATH               An HTTP endpoint to which hook events will be sent to
  --hooks-http-retry NUM          Number of times to retry on a 500 or network timeout [default: 3]
  --hooks-http-backoff SECOND     Number of seconds to wait before retrying each retry [default: 1]
  --hooks-stop-code NUM           Return code from post-receive hook which causes tusd to stop and delete the current upload. A zero value means that no uploads will be stopped [default: 0]
  --expose-metrics                Expose metrics about tusd usage [default: true]
  --metrics-path PATH             Path under which the metrics endpoint will be accessible [default: /metrics]
  --behind-proxy                  Respect X-Forwarded-* and similar headers which may be set by proxies [default: false]
`

func Server() {
  arguments, _ := docopt.ParseDoc(serverUsage)
  cli.Flags.HttpHost, _ = arguments.String("--host")
  cli.Flags.HttpPort, _ = arguments.String("--port")
  cli.Flags.HttpSock, _ = arguments.String("--unix-sock")
  cli.Flags.MaxSize = getInt64(arguments, "--max-size")
  cli.Flags.UploadDir, _ = arguments.String("--dir")
  cli.Flags.StoreSize = getInt64(arguments, "--store-size")
  cli.Flags.Basepath, _ = arguments.String("--base-path")
  cli.Flags.Timeout = getInt64(arguments, "--timeout")
  cli.Flags.S3Bucket, _ = arguments.String("--s3-bucket")
  cli.Flags.S3ObjectPrefix, _ = arguments.String("--s3-object-prefix")
  cli.Flags.S3Endpoint, _ = arguments.String("--s3-endpoint")
  cli.Flags.GCSBucket, _ = arguments.String("--gcs-bucket")
  cli.Flags.GCSObjectPrefix, _ = arguments.String("--gcs-object-prefix")
  cli.Flags.FileHooksDir, _ = arguments.String("--hooks-dir")
  cli.Flags.HttpHooksEndpoint, _ = arguments.String("--hooks-http")
  cli.Flags.HttpHooksRetry, _ = arguments.Int("--hooks-http-retry")
  cli.Flags.HttpHooksBackoff, _ = arguments.Int("--hooks-http-backoff")
  cli.Flags.HooksStopUploadCode, _ = arguments.Int("--hooks-stop-code")
  cli.Flags.PluginHookPath, _ = arguments.String("--hooks-plugin")
  cli.Flags.ShowVersion, _ = arguments.Bool("--version")
  cli.Flags.ExposeMetrics, _ = arguments.Bool("--expose-metrics")
  cli.Flags.MetricsPath, _ = arguments.String("--metrics-path")
  cli.Flags.BehindProxy, _ = arguments.Bool("--behind-proxy")

  dir := cli.Flags.UploadDir
  storeSize := cli.Flags.StoreSize
  maxSize := cli.Flags.MaxSize

  // Attempt to use S3 as a backend if the -s3-bucket option has been supplied.
  // If not, we default to storing them locally on disk.
  Composer := tusd.NewStoreComposer()

  stdout.Printf("Using '%s' as directory storage.\n", dir)
  if err := os.MkdirAll(dir, os.FileMode(0774)); err != nil {
    stderr.Fatalf("Unable to ensure directory exists: %s", err)
  }
  store := filestore.New(dir)
  store.UseIn(Composer)

  if storeSize > 0 {
    limitedstore.New(storeSize, Composer.Core, Composer.Terminater).UseIn(Composer)
    stdout.Printf("Using %.2fMB as storage size.\n", float64(storeSize)/1024/1024)

    // We need to ensure that a single upload can fit into the storage size
    if maxSize > storeSize || maxSize == 0 {
      cli.Flags.MaxSize = storeSize
    }
  }

  stdout.Printf("Using %.2fMB as maximum size.\n", float64(cli.Flags.MaxSize)/1024/1024)

  // Serve
  //if err := SetupPreHooks(Composer); err != nil {
  //  stderr.Fatalf("Unable to setup hooks for handler: %s", err)
  //}

  handler, err := tusd.NewHandler(tusd.Config{
    MaxSize:                 cli.Flags.MaxSize,
    BasePath:                cli.Flags.Basepath,
    RespectForwardedHeaders: cli.Flags.BehindProxy,
    StoreComposer:           Composer,
    NotifyCompleteUploads:   true,
    NotifyTerminatedUploads: true,
    NotifyUploadProgress:    true,
    NotifyCreatedUploads:    true,
  })
  if err != nil {
    stderr.Fatalf("Unable to create handler: %s", err)
  }

  basepath := cli.Flags.Basepath
  address := ""

  if cli.Flags.HttpSock != "" {
    address = cli.Flags.HttpSock
    stdout.Printf("Using %s as socket to listen.\n", address)
  } else {
    address = cli.Flags.HttpHost + ":" + cli.Flags.HttpPort
    stdout.Printf("Using %s as address to listen.\n", address)
  }

  stdout.Printf("Using %s as the base path.\n", basepath)

  //SetupPostHooks(handler)
  //
  //if cli.Flags.ExposeMetrics {
  //  SetupMetrics(handler)
  //  SetupHookMetrics()
  //}

  stdout.Printf(Composer.Capabilities())

  // Do not display the greeting if the tusd handler will be mounted at the root
  // path. Else this would cause a "multiple registrations for /" panic.
  if basepath != "/" {
    // todo: listing page
    //http.HandleFunc("/", DisplayGreeting)
  }
  http.Handle(basepath, http.StripPrefix(basepath, handler))

  var listener net.Listener
  timeoutDuration := time.Duration(cli.Flags.Timeout) * time.Millisecond

  if cli.Flags.HttpSock != "" {
    if listener, err = NewUnixListener(address, timeoutDuration, timeoutDuration); err != nil {
      stderr.Fatalf("Unable to create listener: %s", err)
    }
    stdout.Printf("You can now upload files to: http://%s%s", address, basepath)
  } else {
    if listener, err = NewListener(address, timeoutDuration, timeoutDuration); err != nil {
      stderr.Fatalf("Unable to create listener: %s", err)
    }
  }

  if err = http.Serve(listener, nil); err != nil {
    stderr.Fatalf("Unable to serve: %s", err)
  }
}

func NewListener(addr string, readTimeout, writeTimeout time.Duration) (net.Listener, error) {
  l, err := net.Listen("tcp", addr)
  if err != nil {
    return nil, err
  }

  tl := &Listener{
    Listener:     l,
    ReadTimeout:  readTimeout,
    WriteTimeout: writeTimeout,
  }
  return tl, nil
}

// Binds to a UNIX socket. If the file already exists, try to remove it before
// binding again. This logic is borrowed from Gunicorn
// (see https://github.com/benoitc/gunicorn/blob/a8963ef1a5a76f3df75ce477b55fe0297e3b617d/gunicorn/sock.py#L106)
func NewUnixListener(path string, readTimeout, writeTimeout time.Duration) (net.Listener, error) {
  stat, err := os.Stat(path)

  if err != nil {
    if !os.IsNotExist(err) {
      return nil, err
    }
  } else {
    if stat.Mode()&os.ModeSocket != 0 {
      err = os.Remove(path)

      if err != nil {
        return nil, err
      }
    } else {
      return nil, errors.New("specified path is not a socket")
    }
  }

  l, err := net.Listen("unix", path)

  if err != nil {
    return nil, err
  }

  tl := &Listener{
    Listener:     l,
    ReadTimeout:  readTimeout,
    WriteTimeout: writeTimeout,
  }

  return tl, nil
}

// Listener wraps a net.Listener, and gives a place to store the timeout
// parameters. On Accept, it will wrap the net.Conn with our own Conn for us.
// Original implementation taken from https://gist.github.com/jbardin/9663312
// Thanks! <3
type Listener struct {
  net.Listener
  ReadTimeout  time.Duration
  WriteTimeout time.Duration
}

func (l *Listener) Accept() (net.Conn, error) {
  c, err := l.Listener.Accept()
  if err != nil {
    return nil, err
  }

  //go MetricsOpenConnections.Inc()

  tc := &Conn{
    Conn:         c,
    ReadTimeout:  l.ReadTimeout,
    WriteTimeout: l.WriteTimeout,
  }
  return tc, nil
}

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type Conn struct {
  net.Conn
  ReadTimeout  time.Duration
  WriteTimeout time.Duration

  // closeRecorded will be true if the connection has been closed and the
  // corresponding prometheus counter has been decremented. It will be used to
  // avoid duplicated modifications to this metric.
  closeRecorded bool
}

func (c *Conn) Read(b []byte) (int, error) {
  var err error
  if c.ReadTimeout > 0 {
    err = c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
  } else {
    err = c.Conn.SetReadDeadline(time.Time{})
  }

  if err != nil {
    return 0, err
  }

  return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
  var err error
  if c.WriteTimeout > 0 {
    err = c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
  } else {
    err = c.Conn.SetWriteDeadline(time.Time{})
  }

  if err != nil {
    return 0, err
  }

  return c.Conn.Write(b)
}

func (c *Conn) Close() error {
  // Only decremented the prometheus counter if the Close function has not been
  // invoked before to avoid duplicated modifications.
  if !c.closeRecorded {
    c.closeRecorded = true
    //MetricsOpenConnections.Dec()
  }

  return c.Conn.Close()
}
