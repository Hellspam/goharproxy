package main

import (
	"harproxy"
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


