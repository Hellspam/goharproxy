package harproxy

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

type HarProxy struct {
	// Our go proxy
	Proxy *goproxy.ProxyHttpServer

	// Our port
	Port int

	// Our HAR
	Entries []har.HarEntry
}


type stoppableListener struct {
	net.Listener
	sync.WaitGroup
}

func newStoppableListener(l net.Listener) *stoppableListener {
	return &stoppableListener{l, sync.WaitGroup{}}
}

func NewHarProxy(port int) *HarProxy {
	harEntries := make([]har.HarEntry, 0, 100000)
	harProxy := HarProxy {
		Proxy : createProxy(harEntries),
		Port : port,
		Entries: harEntries,
	}
	return &harProxy
}

func createProxy(harEntries []har.HarEntry) *goproxy.ProxyHttpServer{
	proxy := goproxy.NewProxyHttpServer()
	var before time.Time
	var after time.Time
	tr := transport.Transport{Proxy: transport.ProxyFromEnvironment}
	printed := false

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
	return proxy
}

func (proxy *HarProxy) Start() {
	l, err := net.Listen("tcp", ":" + strconv.Itoa(proxy.Port))
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
	log.Printf("Starting Proxy on port %v\n", proxy.Port)
	http.Serve(sl, proxy.Proxy)
	sl.Wait()
	log.Println("All connections closed - exit")
	for _, v := range proxy.Entries {
		fmt.Printf("Entry\n: %v\n", v.String())
	}
}

