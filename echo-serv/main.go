package main

import (
	"flag"
	"github.com/uniqush/log"
	"net"
	"os"
	"runtime"
)

var argvNrThreads = flag.Int("n", 2, "number of threads")

func serveClient(conn net.Conn) {
	defer conn.Close()

	runtime.GOMAXPROCS(*argvNrThreads)
	var buffer [16]byte
	for {
		n, err := conn.Read(buffer[:])
		if err != nil {
			return
		}
		_, err = conn.Write(buffer[:n])
		if err != nil {
			return
		}
	}
}

func main() {
	addr := "0.0.0.0:8080"
	ln, err := net.Listen("tcp", addr)
	logger := log.NewLogger(os.Stderr, "", log.LOGLEVEL_DEBUG)
	if err != nil {
		logger.Errorf("Error: %v\n", err)
		return
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			logger.Errorf("Error: %v\n", err)
			return
		}
		go serveClient(c)
	}
}
