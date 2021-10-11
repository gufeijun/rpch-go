package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"proxy/local"
	"proxy/remote"

	"golang.org/x/net/proxy"
)

const (
	httpListen   = "127.0.0.1:65500"
	remoteListen = "127.0.0.1:65501"
	localListen  = "127.0.0.1:65502"
)

type HTTPServer struct{}

func (*HTTPServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte(req.RemoteAddr))
}

func startHttpServer() {
	panic(http.ListenAndServe(httpListen, new(HTTPServer)))
}

func sendHttpRequest() string {
	dialer, err := proxy.SOCKS5("tcp", localListen, nil, proxy.Direct)
	if err != nil {
		panic(err)
	}
	client := http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
	}
	resp, err := client.Get(fmt.Sprintf("http://%s", httpListen))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	return string(data)
}

func main() {
	go startHttpServer()
	ch := make(chan string)
	go remote.StartRemoteProxy(remoteListen, ch)
	go local.StartLocalProxy(localListen, remoteListen)
	time.Sleep(time.Second)
	realRequester := sendHttpRequest()
	remoteClient := <-ch
	if remoteClient != realRequester {
		fmt.Printf("addr of real requester should be %s, but is %s\n", remoteClient, realRequester)
		return
	}
	fmt.Println("test succ!!!")
}
