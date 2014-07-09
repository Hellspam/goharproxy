package main

import "harproxy"


func main() {
	harProxy := harproxy.NewHarProxy(9999)
	harProxy.Start()
}


