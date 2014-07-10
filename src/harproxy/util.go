package harproxy

import (
	"net"
	"strconv"
)

func GetPort(l net.Listener) int {
	port, _ := strconv.Atoi(l.Addr().String()[len("[::]:"):])
	return port
}
