package main

import (
	"bytes"
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

func uploadFile(client *gfj.FileServiceClient) error {
	var buff bytes.Buffer
	//这里模拟打开了一个文件
	if _, err := io.WriteString(&buff, "hello world\n"); err != nil {
		return err
	}
	return client.UploadFile(&buff, "uploadedFile.txt")
}

func main() {
	conn, err := rpch.NewClient("127.0.0.1:8080")
	check(err)
	client := gfj.NewFileServiceClient(conn)
	check(uploadFile(client))
	check(writeSomething(client))
	check(readSomething(client))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
