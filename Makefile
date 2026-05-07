SOURCES := $(shell find . -name '*.go')

bin/tabl: $(SOURCES)
	go build -o bin/tabl main.go

bin/tabl-linux_amd64: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/tabl-linux_amd64 main.go

bin/tabl-linux_arm64: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/tabl-linux_arm64 main.go

bin/tabl-macos_amd64: $(SOURCES)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/tabl-macos_amd64 main.go

bin/tabl-macos_arm64: $(SOURCES)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/tabl-macos_arm64 main.go

bin/tabl-windows_amd64.exe: $(SOURCES)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/tabl-windows_amd64.exe main.go


run:
	go run main.go

test:
	go test -v -cover \
		github.com/compgen-io/tabl/bufread \
		github.com/compgen-io/tabl/textfile

clean:
	rm bin/*

.PHONY: run clean test
