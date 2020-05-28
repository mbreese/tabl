all: 
	make -C src

run:
	make -C src run

clean:
	rm -rf bin/tabgo

.PHONY: run clean all
