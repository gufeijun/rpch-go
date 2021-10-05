package main

import (
	"fileserver/gfj"
	"io"
	"os"

	rpch "github.com/gufeijun/rpch-go"
)

type fileService struct{}

func (*fileService) OpenFile(filepath string) (stream io.ReadWriter, onFinish func(), err error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	return file, func() {
		file.Close()
	}, nil
}

func main() {
	svr := rpch.NewServer()
	gfj.RegisterFileService(new(fileService), svr)
	panic(svr.ListenAndServe("tcp", "127.0.0.1:8080"))
}
