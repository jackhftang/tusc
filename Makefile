.PHONY: all
all: get
	GOOS=windows GOARCH=amd64 go build -o tusc_windows_amd64.exe cmd/tusc.go
	GOOS=darwin GOARCH=amd64 go build -o tusc_darwin_amd64 cmd/tusc.go
	GOOS=linux GOARCH=amd64 go build -o tusc_linux_amd64 cmd/tusc.go
	GOOS=linux GOARCH=arm go build -o tusc_linux_arm cmd/tusc.go

.PHONY: get
get:
	go get

