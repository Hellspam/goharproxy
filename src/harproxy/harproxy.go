package harproxy

import (
	"net"
	"net/http"
	"sync"
	"log"
	"har"


	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/transport"

	"time"
	"strconv"
	"io"
	"bytes"
	"strings"
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


func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}


type stoppableListener struct {
	net.Listener
	sync.WaitGroup
}


func newStoppableListener(l net.Listener) *stoppableListener {
	return &stoppableListener{l, sync.WaitGroup{}}
}

func NewHarProxy() *HarProxy {
	return NewHarProxyWithPort(0)
}

func NewHarProxyWithPort(port int) *HarProxy {
	harProxy := HarProxy {
		Proxy : goproxy.NewProxyHttpServer(),
		Port : port,
		Entries: makeNewEntries(),
	}
	return &harProxy
}

func makeNewEntries() []har.HarEntry {
	return make([]har.HarEntry, 0, 100000)
}

func createProxy(proxy *HarProxy) {
	tr := transport.Transport{Proxy: transport.ProxyFromEnvironment}
	proxy.Proxy.Verbose = true
	proxy.Proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		harEntry := new(har.HarEntry)
		harEntry.StartedDateTime = time.Now()
		before := time.Now()
		ctx.RoundTripper = goproxy.RoundTripperFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
			ctx.UserData, resp, err = tr.DetailedRoundTrip(req)
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
	log.Printf("Starting harproxy server on port :%v", GetPort(l))
	go http.Serve(proxy.StoppableListener, proxy.Proxy)
	log.Printf("Stared harproxy server on port :%v", GetPort(l))
}

func (proxy *HarProxy) Stop() {
	log.Printf("Stopping harproxy server on port :%v", proxy.Port)
	proxy.StoppableListener.Add(1)
	proxy.StoppableListener.Close()
	proxy.StoppableListener.Done()
}

func (proxy *HarProxy) ClearEntries() {
	log.Printf("Clearing HAR for harproxy server on port :%v", proxy.Port)
	proxy.Entries = makeNewEntries()
}

func (proxy *HarProxy) NewHarReader() io.Reader {
	var buffer bytes.Buffer
	for _, entry := range proxy.Entries {
		buffer.WriteString(entry.String())
	}
	return strings.NewReader(buffer.String())
}

