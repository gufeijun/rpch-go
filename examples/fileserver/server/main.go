package main

import (
	"io"
	"os"

	"github.com/gufeijun/rpch-go/examples/fileserver/gfj"

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

func (*fileService) UploadFile(r io.Reader, filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, r)
	return err
}

func main() {
	svr := rpch.NewServer()
	gfj.RegisterFileService(new(fileService), svr)
	panic(svr.ListenAndServe("127.0.0.1:8080"))
}
