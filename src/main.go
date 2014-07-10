package main

import (
	"harproxy"
	"time"
)


func main() {
	harProxy := harproxy.NewHarProxy(9999)
	go func() {harProxy.Start()}()
	time.Sleep(5 * time.Second)
	harProxy.Stop()
	harProxy.PrintEntries()
}


