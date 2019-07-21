# tusc

![build status](https://travis-ci.com/jackhftang/tusc.svg?branch=master)

A single binary for both server and client of [tus resumable upload protocol](https://tus.io)

### Quick start on local  

Start server 

```bash
$ tusc server -h 127.0.0.1 -p 8080
```

Create and upload a file 

```bash 
$ echo test > test.txt
$ tusc client http://127.0.0.1:8080 text.txt  
```

## Server  

The implementation is a wrapper of [tusd](https://github.com/tus/tusd)

## Client

The implementation is a wrapper of [go-tusd](https://github.com/eventials/go-tus)
