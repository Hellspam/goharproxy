package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"log"
	"har"


	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/transport"

	"time"
)

type stoppableListener struct {
	net.Listener
	sync.WaitGroup
}

func newStoppableListener(l net.Listener) *stoppableListener {
	return &stoppableListener{l, sync.WaitGroup{}}
}

func orPanic(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	var before time.Time
	var after time.Time
	tr := transport.Transport{Proxy: transport.ProxyFromEnvironment}
	printed := false
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		harEntry := har.HarEntry{}
		harEntry.StartedDateTime = time.Now()
		ctx.RoundTripper = goproxy.RoundTripperFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
			ctx.UserData, resp, err = tr.DetailedRoundTrip(req)
			harResponse := har.ParseResponse(resp)
			harEntry.Response = harResponse
			after = time.Now()
			harEntry.Time = after.Sub(before).Nanoseconds() / 1e6
			fmt.Printf("Entry\n: %v\n", harEntry.String())
			printed = true
			return
		})
		before = time.Now()
		harRequest := har.ParseRequest(req)
		harEntry.Request = harRequest

		return req, nil
	})

	l, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("listen:", err)
	}
	http.ListenAndServe(":9999",proxy )
	sl := newStoppableListener(l)
	http.Serve(sl, proxy)
	sl.Wait()
}
