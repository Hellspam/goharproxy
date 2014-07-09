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
	"os"
	"os/signal"
	"strconv"
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
	createProxy(9999)
}

func createProxy(port int) {
	proxy := goproxy.NewProxyHttpServer()
	var before time.Time
	var after time.Time
	tr := transport.Transport{Proxy: transport.ProxyFromEnvironment}
	printed := false
	harEntries := make([]har.HarEntry, 0, 100000)
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		harEntry := new(har.HarEntry)
		harEntry.StartedDateTime = time.Now()
		ctx.RoundTripper = goproxy.RoundTripperFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
			ctx.UserData, resp, err = tr.DetailedRoundTrip(req)
			harResponse := har.ParseResponse(resp)
			harEntry.Response = harResponse
			after = time.Now()
			harEntry.Time = after.Sub(before).Nanoseconds() / 1e6
			if !printed {
				fmt.Printf("Entry\n: %v\n", harEntry.String())
				printed = true
			}
			harEntries = append(harEntries, *harEntry)
			return
		})
		before = time.Now()
		harRequest := har.ParseRequest(req)
		harEntry.Request = harRequest
		ipaddr, _ := net.LookupIP(req.URL.Host)
		harEntry.ServerIpAddress = ipaddr[0].String()

		return req, nil
	})

	l, err := net.Listen("tcp", ":" + strconv.Itoa(port))
	if err != nil {
		log.Fatal("listen:", err)
	}
	sl := newStoppableListener(l)
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		log.Println("Got SIGINT exiting")
		sl.Add(1)
		sl.Close()
		sl.Done()
	}()
	log.Printf("Starting Proxy on port %v\n", port)
	http.Serve(sl, proxy)
	sl.Wait()
	log.Println("All connections closed - exit")
	for _, v := range harEntries {
		fmt.Printf("Entry\n: %v\n", v.String())
	}
}
