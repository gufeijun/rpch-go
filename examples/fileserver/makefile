EXECUTABLES				:= hgen go
T 						:= $(foreach exec,$(EXECUTABLES),\
        				$(if $(shell which $(exec)),,$(error "No $(exec) in PATH!")))

.PHONY:all
all:bin gfj bin/server bin/client
	@echo all binaries are in directory: bin

bin/server:server/main.go gfj 
	go build -o $@ $<

bin/client:client/main.go gfj 
	go build -o $@ $<
	
gfj:
	hgen -lang go file.gfj
bin:
	@mkdir bin
.PHONY:clean
clean:
	-rm bin -rf
	-rm gfj -rf
