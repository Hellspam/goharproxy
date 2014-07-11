package main

import (
	"harproxy"
	"log"
	"net/http"
)
import (
	_ "net/http/pprof"
	"flag"
)


func main() {
	port := flag.Int("p", 8080, "Port to listen on")
	flag.Parse()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	harproxy.NewProxyServer(*port)
}


