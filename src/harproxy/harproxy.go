package harproxy

import (
	"net"
	"net/http"
	"sync"
	"log"
	"time"
	"strconv"
	"io"
	"strings"
	"regexp"
	"fmt"
	"encoding/json"


	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/transport"
)

// HarProxy

type HarProxy struct {
	// Our go proxy
	Proxy *goproxy.ProxyHttpServer

	// The port our proxy is listening on
	Port int

	// Our HAR log.
	// Starting size of 1000 entries, enlarged if necessary
	// Read the specification here: http://www.softwareishard.com/blog/har-12-spec/
	HarLog *HarLog

	// Stoppable listener - used to stop http proxy
	StoppableListener *stoppableListener

	// This channel is used to signal when the http.Serve function is done serving our proxy
	isDone chan bool
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
		HarLog : newHarLog(),
	}
	createProxy(&harProxy)
	return &harProxy
}


func createProxy(proxy *HarProxy) {
	tr := transport.Transport{Proxy: transport.ProxyFromEnvironment}
	proxy.Proxy.Verbose = true
	proxy.Proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		harEntry := new(HarEntry)
		harEntry.StartedDateTime = time.Now()
		before := time.Now()
		ctx.RoundTripper = goproxy.RoundTripperFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
			ctx.UserData, resp, err = tr.DetailedRoundTrip(req)
			harResponse := parseResponse(resp)
			harEntry.Response = harResponse
			after := time.Now()
			harEntry.Time = after.Sub(before).Nanoseconds() / 1e6
			proxy.HarLog.addEntry(*harEntry)
			return
		})
		harRequest := parseRequest(req)
		harEntry.Request = harRequest
		if ip, _, err := net.ParseCIDR(req.URL.Host); err == nil {
			harEntry.ServerIpAddress = string(ip)
		}

		if ipaddr, err := net.LookupIP(req.URL.Host); err == nil {
			harEntry.ServerIpAddress = ipaddr[0].String()
		}


		return req, nil
	})
}

func (proxy *HarProxy) Start() {
	l, err := net.Listen("tcp", ":" + strconv.Itoa(proxy.Port))
	if err != nil {
		log.Fatal("listen:", err)
	}
	proxy.StoppableListener = newStoppableListener(l)
	proxy.Port = GetPort(l)
	log.Printf("Starting harproxy server on port :%v", proxy.Port)
	proxy.isDone = make(chan bool)
	go func() {
		http.Serve(proxy.StoppableListener, proxy.Proxy)
		log.Printf("Done serving proxy on port: %v", proxy.Port)
		proxy.isDone <- true
	}()
	log.Printf("Stared harproxy server on port :%v", proxy.Port)
}

func (proxy *HarProxy) Stop() {
	log.Printf("Stopping harproxy server on port :%v", proxy.Port)
	proxy.StoppableListener.Add(1)
	proxy.StoppableListener.Close()
	<-proxy.isDone
	proxy.StoppableListener.Done()
	proxy = nil
}

func (proxy *HarProxy) ClearEntries() {
	log.Printf("Clearing HAR for harproxy server on port :%v", proxy.Port)
	proxy.HarLog.Entries = makeNewEntries()
}

func (proxy *HarProxy) NewHarReader() io.Reader {
	str, _ := json.Marshal(proxy.HarLog)
	return strings.NewReader(string(str))
}

//

// HarProxyServer

var portAndProxy map[int]*HarProxy = make(map[int]*HarProxy, 5000)

var portPathRegex *regexp.Regexp = regexp.MustCompile("/(\\d*)(/.*)?")

type ProxyServerPort struct {
	Port int   `json:"port"`
}

type ProxyServerErr struct {
	Error string	`json:"error"`
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/proxy"):]
	method := r.Method

	log.Printf("PATH:[%v]\n", r.URL.Path)
	log.Printf("FILTERED:[%v]\n", path)
	log.Printf("METHOD:[%v]\n", method)
	var harProxy *HarProxy
	var port int
	if portPathRegex.MatchString(path) {
		portStr := portPathRegex.FindStringSubmatch(path)[1]
		port, _ = strconv.Atoi(portStr)
		if portAndProxy[port] == nil {
			w.WriteHeader(http.StatusNotFound)
			errMsg := fmt.Sprintf("No such port:[%v]", port)
			log.Printf(errMsg)
			err := ProxyServerErr {
				Error : errMsg,
			}
			json.NewEncoder(w).Encode(&err)
			return
		}
		harProxy = portAndProxy[port]
		log.Printf("PORT:[%v]\n", port)
	}
	switch {
	case path == "" && method == "POST":
		log.Println("MATCH CREATE")
		createNewHarProxy(w)
	case strings.HasSuffix(path, "har") && method == "PUT":
		log.Println("MATCH PRINT")
		getHarLog(harProxy, w)
	case portPathRegex.MatchString(path) && method == "DELETE":
		log.Println("MATCH DELETE")
		deleteHarProxy(port, w)
	}

}

func deleteHarProxy(port int, w http.ResponseWriter) {
	log.Printf("Deleting proxy on port :%v\n", port)
	harProxy := portAndProxy[port]
	harProxy.Stop()
	delete(portAndProxy, port)
	harProxy = nil
	w.WriteHeader(http.StatusOK)

}

func getHarLog(harProxy *HarProxy, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(harProxy.HarLog)
	harProxy.ClearEntries()

}

func createNewHarProxy(w http.ResponseWriter) {
	log.Printf("Got request to start new proxy\n")
	harProxy := NewHarProxy()
	harProxy.Start()
	port := GetPort(harProxy.StoppableListener.Listener)
	harProxy.Port = port

	portAndProxy[port] = harProxy

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	proxyServerPort := ProxyServerPort {
		Port : port,
	}
	json.NewEncoder(w).Encode(&proxyServerPort)
}


func NewProxyServer(port int) {
	http.HandleFunc("/proxy", proxyHandler)
	http.HandleFunc("/proxy/", proxyHandler)
	log.Printf("Started HAR Proxy server on port :%v, Waiting for proxy start request\n", port)
	log.Fatal(http.ListenAndServe(":" + strconv.Itoa(port), nil))
}
