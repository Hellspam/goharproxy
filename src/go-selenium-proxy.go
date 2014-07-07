package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"log"
	"har"


	"github.com/elazarl/goproxy"
//	"github.com/elazarl/goproxy/transport"

	"time"
)

type stoppableListener struct {
	net.Listener
	sync.WaitGroup
}

func newStoppableListener(l net.Listener) *stoppableListener {
	return &stoppableListener{l, sync.WaitGroup{}}
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	var before time.Time
	var after time.Time
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		before = time.Now()
		harRequest := har.ParseRequest(req)
		fmt.Printf("Request: %v\n" , harRequest.String())
		return req, nil
	})
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		after = time.Now()
		fmt.Printf("Total time %v %v: %v\n", before, after, after.Sub(before))
		return resp
	})
	l, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("listen:", err)
	}
	sl := newStoppableListener(l)
	http.Serve(sl, proxy)
	sl.Wait()
}
