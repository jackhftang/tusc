package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/eventials/go-tus"
	"github.com/eventials/go-tus/leveldbstore"
	"github.com/tus/tusd/cmd/tusd/cli"
	"net/http"
	"os"
	"strconv"
)

const Version = "0.1.0"

const usage = `Usage:
  tusc (server|s) [options]
  tusc (client|c) <file> [<url>] [options]
  tusc --help
  tusc --version`

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

func exitWithMessage(msg ...interface{}) {
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

func server() {
	arguments, _ := docopt.ParseArgs(serverUsage, nil, Version)
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
	cli.CreateComposer()
	cli.Serve()
}

func client() {
	arguments, _ := docopt.ParseArgs(clientUsage, nil, Version)

	file, _ := arguments.String("<file>")
	url, _ := arguments.String("<url>")
	resume := getBool(arguments, "--resumable")

	f, err := os.Open(file)
	if err != nil {
		exitWithMessage("Cannot open file: " + file)
	}
	defer f.Close()

	// create the tus client
	var store tus.Store
	if resume {
		path := getString(arguments, "--store")
		store, err = leveldbstore.NewLeveldbStore(path)
		if err != nil {
			exitWithMessage("Cannot Open "+path, clientUsage)
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

	fmt.Printf(uploader.Url())

	// start the uploading process.
	if err = uploader.Upload(); err != nil {
		exitWithMessage("Upload incomplete")
	}
}

func main() {
	if len(os.Args) < 2 {
		exitWithMessage("No command", usage)
	}
	switch cmd := os.Args[1]; cmd {
	case "server", "s":
		server()
	case "client", "c":
		client()
	default:
		exitWithMessage("Unknown command: "+cmd, usage)
	}
}
