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
	"strconv"
)

type HarProxy struct {
	// Our go proxy
	Proxy *goproxy.ProxyHttpServer

	// Our port
	Port int

	// Our HAR entries channel
	Entries []har.HarEntry

	// Stoppable listner - used to stop http proxy
	StoppableListener *stoppableListener
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
		Proxy : goproxy.NewProxyHttpServer(),
		Port : port,
		Entries: harEntries,
	}
	return &harProxy
}

func createProxy(proxy *HarProxy) {
	tr := transport.Transport{Proxy: transport.ProxyFromEnvironment}
	proxy.Proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		harEntry := new(har.HarEntry)
		harEntry.StartedDateTime = time.Now()
		before := time.Now()
		ctx.RoundTripper = goproxy.RoundTripperFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
			ctx.UserData, resp, err = tr.DetailedRoundTrip(req)
			fmt.Println("Doing stuff")
			harResponse := har.ParseResponse(resp)
			harEntry.Response = harResponse
			after := time.Now()
			harEntry.Time = after.Sub(before).Nanoseconds() / 1e6
			proxy.Entries = append(proxy.Entries, *harEntry)
			return
		})
		harRequest := har.ParseRequest(req)
		harEntry.Request = harRequest
		ipaddr, _ := net.LookupIP(req.URL.Host)
		harEntry.ServerIpAddress = ipaddr[0].String()

		return req, nil
	})
}

func (proxy *HarProxy) Start() {
	createProxy(proxy)
	l, err := net.Listen("tcp", ":" + strconv.Itoa(proxy.Port))
	if err != nil {
		log.Fatal("listen:", err)
	}
	proxy.StoppableListener = newStoppableListener(l)
	log.Printf("Starting harproxy server on port :%v", proxy.Port)
	http.Serve(proxy.StoppableListener, proxy.Proxy)
	proxy.StoppableListener.Wait()
}

func (proxy *HarProxy) Stop() {
	log.Printf("Stopping harproxy server on port :%v", proxy.Port)
	proxy.StoppableListener.Add(1)
	proxy.StoppableListener.Close()
	proxy.StoppableListener.Done()
}

func (proxy *HarProxy) PrintEntries() {
	for _, entry := range proxy.Entries {
		fmt.Printf("Entry\n: %v\n", entry.Request.Url)
	}
}

