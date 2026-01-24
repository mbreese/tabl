SOURCES := $(shell find . -name '*.go')

bin/tabl.linux_amd64: bin/tabl
	GOOS=linux GOARCH=amd64 go build -o bin/tabl.linux_amd64 main.go

bin/tabl.linux_arm64: bin/tabl
	GOOS=linux GOARCH=arm64 go build -o bin/tabl.linux_aarm64 main.go

bin/tabl.macos_amd64: bin/tabl
	GOOS=darwin GOARCH=amd64 go build -o bin/tabl.macos_amd64 main.go

bin/tabl.macos_arm64: bin/tabl
	GOOS=darwin GOARCH=amd64 go build -o bin/tabl.macos_arm64 main.go

bin/tabl.exe: bin/tabl
	GOOS=windows GOARCH=amd64 go build -o bin/tabl.exe main.go


run:
	go run main.go

test:
	go test -v -cover \
		github.com/mbreese/tabl/bufread \
		github.com/mbreese/tabl/textfile

clean:
	rm bin/*

.PHONY: run clean test
