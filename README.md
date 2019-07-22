# tusc

[![build status](https://travis-ci.com/jackhftang/tusc.svg?branch=master)](https://travis-ci.org/jackhftang/tusc)

A single binary for both server and client of [tus resumable upload protocol](https://tus.io). 

This is a wrapper of [tusd](https://github.com/tus/tusd) with nginx-like file listing page (or index page) is added. 
Features like S3, GCS, Prometheus, Hooks are removed from tusd, in favor of smaller binary size (< 5 MB after upx-ed rather than > 30MB raw). 
This is a command line implementation this library [go-tusd](https://github.com/eventials/go-tus).

## Quick Start  

Visit [releases page](https://github.com/jackhftang/tusc/releases) and download the tusc binary. 

```bash 
$ curl -LO https://github.com/jackhftang/tusc/releases/download/<version>/tusc_linux_amd64 -o tusc
$ chmod u+x tusc 
```

Start server 

```bash
$ tusc server -h 127.0.0.1 -p 8080
```

Create and upload a file 

```bash 
$ echo test > test.txt
$ tusc client http://127.0.0.1:8080/files/ text.txt     # not resumable
$ tusc client http://127.0.0.1:8080/files/ text.txt -r  # resumable
```

And then visit to [file listing page](http://127.0.0.1:8080)

### Server Options 


```
$ tusc s --help
tusc server

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
```

### Client Options

```
$ tusc c --help
tusc client

Usage:
  tusc (client|c) <url> <file> [options]
  tusc (client|c) --help

Options:
  -r --resumable            Save meta data in store for resumable uploads
  --store PATH              Path to save meta data for resume [default: ./.tusc]
  --chuck-size BYTE         Size of chucks of file [default: 2097152]
  --override-patch-method   Sending a POST request instead of PATCH [default: false] 
```
