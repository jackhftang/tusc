.PHONY: all
all: clean get
	GOOS=windows GOARCH=amd64 go build -ldflags "-w -s" -o tusc_windows_amd64.exe cmd/tusc.go
	GOOS=darwin GOARCH=amd64 go build -ldflags "-w -s" -o tusc_darwin_amd64 cmd/tusc.go
	GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o tusc_linux_amd64 cmd/tusc.go
	GOOS=linux GOARCH=arm go build -ldflags "-w -s" -o tusc_linux_arm cmd/tusc.go
	which upx && upx tusc* || true

.PHONY: get
get:
	go get

build:
	go build cmd/tusc.go

clean:
	rm -rf tusc* data .tusc

release-patch: all
	release-it -n -i patch

release-minor: all
	release-it -n -i minor

release-major: all
	release-it -n -i major
