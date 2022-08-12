build_flags = -ldflags "-extldflags=-static -w -s"

.PHONY: release test linux-amd64 linux-386 linux-arm linux-arm64 macos-amd64 macos-arm64 windows-amd64 windows-386
.DEFAULT_GOAL := test

release: linux-amd64 linux-386 linux-arm linux-arm64 macos-amd64 macos-arm64 windows-amd64 windows-386

linux-amd64:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build  $(build_flags) -o release/doh-relay_linux-amd64 .
linux-386:
	mkdir -p release
	GOOS=linux GOARCH=386 go build  $(build_flags) -o release/doh-relay_linux-386 .
linux-arm:
	mkdir -p release
	GOOS=linux GOARCH=arm go build  $(build_flags) -o release/doh-relay_linux-arm .
linux-arm64:
	mkdir -p release
	GOOS=linux GOARCH=arm64 go build  $(build_flags) -o release/doh-relay_linux-arm64 .
macos-amd64:
	mkdir -p release
	GOOS=darwin GOARCH=amd64 go build $(build_flags) -o release/doh-relay_macos-amd64 .
macos-arm64:
	mkdir -p release
	GOOS=darwin GOARCH=arm64 go build $(build_flags) -o release/doh-relay_macos-arm64 .
windows-amd64:
	mkdir -p release
	GOOS=windows GOARCH=amd64 go build $(build_flags) -o release/doh-relay_windows-amd64.exe .
windows-386:
	mkdir -p release
	GOOS=windows GOARCH=386 go build $(build_flags) -o release/doh-relay_windows-386.exe .

test:
	go test -v ./
