package main

import (
	"github.com/hellspam/go-selenium-proxy/src/harproxy"
	"log"
	"net/http"
)
import _ "net/http/pprof"


func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	harproxy.NewProxyServer(8080)
}


