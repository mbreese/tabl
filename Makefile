SOURCES := $(shell find . -name '*.go')

bin/tabl: go.mod go.sum $(SOURCES) 
	go build -o bin/tabl main.go

run:
	go run main.go

test:
	go test -v -cover \
		github.com/mbreese/tabl/bufread \
		github.com/mbreese/tabl/textfile

clean:
	rm bin/*

.PHONY: run clean test
