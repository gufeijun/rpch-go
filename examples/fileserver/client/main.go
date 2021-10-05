package main

import (
	"fileserver/gfj"
	"io"
	"os"

	"github.com/gufeijun/rpch-go"
)

func writeSomething(client *gfj.FileServiceClient) error {
	file, err := client.OpenFile("test.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte("hello world\n"))
	if err != nil {
	}
	return err
}

func readSomething(client *gfj.FileServiceClient) error {
	file, err := client.OpenFile("test.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	io.Copy(os.Stdout, file)
	return err

}

func main() {
	conn, err := rpch.NewClient("127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
	client := gfj.NewFileServiceClient(conn)
	if err := writeSomething(client); err != nil {
		panic(err)
	}
	if err := readSomething(client); err != nil {
		panic(err)
	}
}
