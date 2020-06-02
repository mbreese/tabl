SOURCES := $(shell find . -name '*.go')

bin/tabgo: go.mod go.sum $(SOURCES) 
	go build -o bin/tabgo main.go

run:
	go run main.go

test:
	go test -v -cover \
		github.com/mbreese/tabgo/bufread \
		github.com/mbreese/tabgo/textfile

clean:
	rm bin/*

.PHONY: run clean test
