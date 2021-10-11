EXECUTABLES				:= hgen go
T 						:= $(foreach exec,$(EXECUTABLES),\
        				$(if $(shell which $(exec)),,$(error "No $(exec) in PATH!")))

.PHONY:all
all:gfj app

app:main.go
	go build -o app main.go
	
gfj:
	hgen math.gfj
.PHONY:clean
clean:
	-rm gfj -rf
	-rm app
