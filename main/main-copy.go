package main

import (
	"flag"
	
	"github.com/Hellspam/goharproxy"
	"os"
	"log"
	"runtime/pprof"
	"time"
	"runtime"
)


func main() {
	var memprofile = flag.String("memprofile", "", "write memory profile to this file")
	flag.Parse()

	proxy := goharproxy.NewHarProxyWithPort(9999)
	writeHeapProfile(*memprofile + "before_start.pprof")
	proxy.Start()
	writeHeapProfile(*memprofile + "after_start.pprof")
	time.Sleep(60 * time.Second)
	writeHeapProfile(*memprofile + "before_clear.pprof")
	proxy.ClearEntries()
	writeHeapProfile(*memprofile + "after_clear.pprof")
	proxy.Stop()
	writeHeapProfile(*memprofile + "after_stop.pprof")
	return
}

func writeHeapProfile(name string) {
	if name != "" {
		f, err := os.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("CREATING MEMFILE")
		pprof.WriteHeapProfile(f)
		runtime.GC()
		f.Close()
		return
	}
}


