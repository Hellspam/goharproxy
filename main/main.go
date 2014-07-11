package main

import (
	"flag"
	
	"github.com/Hellspam/goharproxy"
//	_ "net/http/pprof"
)


func main() {
	port := flag.Int("p", 8080, "Port to listen on")
	verbose := flag.Bool("v", true, "Verbosity")
	flag.Parse()
//	go func() {
//		log.Println(http.ListenAndServe("localhost:6060", nil))
//	}()
	goharproxy.Verbosity = *verbose
	goharproxy.NewProxyServer(*port)
}


