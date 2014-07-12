package goharproxy

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


	"github.com/Hellspam/goproxy"
	"github.com/Hellspam/goproxy/transport"
)

// HarProxy

var Verbosity bool

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

	// Stores hosts we want to redirect to a different ip / host
	hostEntries []ProxyHosts
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
		Proxy 		: goproxy.NewProxyHttpServer(),
		Port 		: port,
		HarLog 		: newHarLog(),
		hostEntries : make([]ProxyHosts, 0, 100),
		isDone 		: make(chan bool),
	}
	createProxy(&harProxy)
	return &harProxy
}


func createProxy(proxy *HarProxy) {
	tr := transport.Transport{Proxy: transport.ProxyFromEnvironment}
	proxy.Proxy.Verbose = Verbosity
	proxy.Proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		harEntry := new(HarEntry)
		harEntry.StartedDateTime = time.Now()
		before := time.Now()
		ctx.RoundTripper = goproxy.RoundTripperFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
			ctx.UserData, resp, err = tr.DetailedRoundTrip(req)
			if err != nil {
				return resp, err
			}
			resp, err = handleResponse(resp, harEntry, proxy)
			after := time.Now()
			harEntry.Time = after.Sub(before).Nanoseconds() / 1e6
			proxy.HarLog.addEntry(*harEntry)
			return
		})
		return handleRequest(req, harEntry, proxy)
	})
}

func handleRequest(req *http.Request, harEntry *HarEntry, harProxy *HarProxy) (*http.Request, *http.Response) {
	harRequest := parseRequest(req)
	harEntry.Request = harRequest
	replaceHost(req, harProxy)
	fillIpAddress(req, harEntry)

	return req, nil
}

func replaceHost(req *http.Request, harProxy *HarProxy) {
	for _, hostEntry := range harProxy.hostEntries {
		if req.URL.Host == hostEntry.Host {
			req.URL.Host = hostEntry.NewHost
			return
		}
	}
}

func handleResponse(resp *http.Response, harEntry *HarEntry, harProxy *HarProxy) (newResp *http.Response, err error) {
	harResponse := parseResponse(resp)
	harEntry.Response = harResponse

	return resp, nil
}

func fillIpAddress(req *http.Request, harEntry *HarEntry) {
	if ip, _, err := net.ParseCIDR(req.URL.Host); err == nil {
		harEntry.ServerIpAddress = string(ip)
	}

	if ipaddr, err := net.LookupIP(req.URL.Host); err == nil {
		harEntry.ServerIpAddress = ipaddr[0].String()
	}
}

func (proxy *HarProxy) AddHostEntries(hostEntries []ProxyHosts) {
	entries := proxy.hostEntries
	m := len(entries)
	n := m + len(hostEntries)
	if n > cap(entries) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newEntries := make([]ProxyHosts, (n+1)*2)
		copy(newEntries, entries)
		entries = newEntries
	}
	entries = entries[0:n]
	copy(entries[m:n], hostEntries)
	proxy.hostEntries = entries
}

func (proxy *HarProxy) Start() {
	l, err := net.Listen("tcp", ":" + strconv.Itoa(proxy.Port))
	if err != nil {
		log.Fatal("listen:", err)
	}
	proxy.StoppableListener = newStoppableListener(l)
	proxy.Port = GetPort(l)
	log.Printf("Starting harproxy server on port :%v", proxy.Port)
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

type ProxyServerMessage struct {
	Message string 		`json:"message"`
}

type ProxyHosts struct {
	Host 	string 		`json:"host"`
	NewHost string		`json:"NewHost"`
}

func addHostEntries(harProxy *HarProxy, r *http.Request, w http.ResponseWriter) {
	hostEntries := make([]ProxyHosts, 0, 10)
	err := json.NewDecoder(r.Body).Decode(&hostEntries)
	if err != nil {
		writeErrorMessage(w, http.StatusInternalServerError,  err.Error())
		return
	}

	harProxy.AddHostEntries(hostEntries)
	writeMessage(w, "Added hosts entries successfully")
}

func deleteHarProxy(port int, w http.ResponseWriter) {
	log.Printf("Deleting proxy on port :%v\n", port)
	harProxy := portAndProxy[port]
	harProxy.Stop()
	delete(portAndProxy, port)
	harProxy = nil
	writeMessage(w, fmt.Sprintf("Deleted proxy for port [%v] succesfully", port))
}

func getHarLog(harProxy *HarProxy, w http.ResponseWriter) {
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
	proxyServerPort := ProxyServerPort {
		Port : port,
	}
	json.NewEncoder(w).Encode(&proxyServerPort)
}

func getProxyForPath(path string, w http.ResponseWriter) (*HarProxy, string) {
	if portPathRegex.MatchString(path) {
		portStr := portPathRegex.FindStringSubmatch(path)[1]
		port, _ := strconv.Atoi(portStr)
		if portAndProxy[port] == nil {
			writeErrorMessage(w, http.StatusNotFound, fmt.Sprintf("No proxy for port [%v]", port))
			return nil, path
		}

		log.Printf("PORT:[%v]\n", port)
		return portAndProxy[port],  path[len("/" + portStr):]
	}

	return nil,path
}

func writeMessage(w http.ResponseWriter, msg string) {
	w.Header().Add("Content-type", "application/json")
	proxyMessage := ProxyServerMessage {
		Message : msg,
	}
	json.NewEncoder(w).Encode(&proxyMessage)
}

func writeErrorMessage(w http.ResponseWriter, httpStatus int,  msg string) {
	log.Printf("ERROR :[%v]", msg)
	w.WriteHeader(httpStatus)
	errorMessage := ProxyServerErr {
		Error : msg,
	}
	json.NewEncoder(w).Encode(&errorMessage)
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, "/proxy") {
		errHandler(w, r)
		return
	}
	path := r.URL.Path[len("/proxy"):]
	method := r.Method

	log.Printf("PATH:[%v]\n", r.URL.Path)
	log.Printf("FILTERED:[%v]\n", path)
	log.Printf("METHOD:[%v]\n", method)
	if path == "" && method == "POST" {
		log.Println("MATCH CREATE")
		createNewHarProxy(w)
		return
	}

	harProxy, path := getProxyForPath(path, w)
	switch {
	case harProxy == nil:
		return
	case strings.HasSuffix(path, "har") && method == "PUT":
		log.Println("MATCH PRINT")
		getHarLog(harProxy, w)
	case path == "" && method == "DELETE":
		log.Println("MATCH DELETE")
		deleteHarProxy(harProxy.Port, w)
	case strings.HasSuffix(path, "hosts") && method == "POST":
		log.Println("MATCH HOSTS")
		addHostEntries(harProxy, r, w)
	default:
		log.Printf("No such path: [%v]", path)
		writeErrorMessage(w, http.StatusNotFound, fmt.Sprintf("No such path [%s] with method %v" , path, method))
	}
}

func errHandler(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("No such path: [%v]", r.URL.Path)
	log.Println(msg)
	writeErrorMessage(w, http.StatusNotFound, msg)
}

func NewProxyServer(port int) {
	http.HandleFunc("/", errHandler)
	http.HandleFunc("/proxy", proxyHandler)
	http.HandleFunc("/proxy/", proxyHandler)

	log.Printf("Started HAR Proxy server on port :%v, Waiting for proxy start request\n", port)
	log.Fatal(http.ListenAndServe(":" + strconv.Itoa(port), nil))
}
