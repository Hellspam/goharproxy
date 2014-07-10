package harproxy

import (
	"log"
	"net/http"
	"regexp"
	"io/ioutil"
	"strings"
	"fmt"
	"strconv"
)

var portAndProxy map[int]*HarProxy = make(map[int]*HarProxy, 100000)

var portPathRegex *regexp.Regexp = regexp.MustCompile("/(\\d.*)(/.*)?")

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
			fmt.Fprint(w, "No such port")
			return
		}
		harProxy = portAndProxy[port]
		log.Printf("PORT:[%v]\n", port)
		log.Printf("PROXY:[%v]\n", harProxy)
	}
	switch {
	case path == "" && method == "POST":
		createNewHarProxy(w)
	case strings.HasSuffix(path, "har") && method == "PUT":
		log.Printf("MATCH PRINT")
		printHarEntries(harProxy, w)
	case portPathRegex.MatchString(path) && method == "DELETE":
		log.Printf("MATCH DELETE")
		deleteHarProxy(port, w)
	}

}

func deleteHarProxy(port int, w http.ResponseWriter) {
	log.Printf("Deleting proxy on port :%v\n", port)
	harProxy := portAndProxy[port]
	harProxy.Stop()
	delete(portAndProxy, port)
	w.WriteHeader(http.StatusOK)
}

func printHarEntries(harProxy *HarProxy, w http.ResponseWriter) {
	bytes, err := ioutil.ReadAll(harProxy.NewHarReader())
	if err != nil {
		log.Fatal(err)
	}
	harProxy.ClearEntries()
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)

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
	fmt.Fprintf(w, "{\"port\" : %v}", port)
}


func NewProxyServer(port int) {
	http.HandleFunc("/proxy", proxyHandler)
	http.HandleFunc("/proxy/", proxyHandler)
	log.Printf("Started HAR Proxy server on port :%v, Waiting for proxy start request\n", port)
	log.Fatal(http.ListenAndServe(":" + strconv.Itoa(port), nil))
}
