SOURCES := $(shell find . -name '*.go')

../bin/tabgo: $(SOURCES) 
	go build -o ../bin/tabgo main.go

run:
	go run main.go


.PHONY: run
