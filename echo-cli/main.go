package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

func pingPong(conn net.Conn, m int, buf []byte) (d time.Duration, err error) {
	var n int
	var b [16]byte
	start := time.Now()
	for i := 0; i < m; i++ {
		n, err = conn.Write(buf[:])
		if err != nil {
			return
		}
		_, err = io.ReadFull(conn, b[:n])
		if err != nil {
			return
		}
	}
	d = time.Since(start)
	if !bytes.Equal(buf[:n], b[:n]) {
		err = fmt.Errorf("Wrong content")
		return
	}
	return
}

type result struct {
	d   time.Duration
	err error
}

func Client(addr string, buf []byte, n int, start <-chan bool, stop <-chan bool, resChan chan<- *result, wg *sync.WaitGroup) {
	defer wg.Done()
	<-start
	res := new(result)
	var conn net.Conn
	conn, res.err = net.Dial("tcp", addr)
	if res.err != nil {
		resChan <- res
		return
	}
	defer conn.Close()
	res.d, res.err = pingPong(conn, n, buf)
	resChan <- res
}

type BenchClient struct {
	N       int
	M       int
	Addr    string
	start   chan bool
	stop    chan bool
	resChan chan *result
	out     io.Writer
	wg      *sync.WaitGroup
}

func (self *BenchClient) Connect() error {
	if self.start == nil {
		self.start = make(chan bool)
	}
	if self.stop == nil {
		self.stop = make(chan bool)
	}
	if self.resChan == nil {
		self.resChan = make(chan *result)
	}
	if self.M <= 0 {
		self.M = 1
	}
	var buf [16]byte
	_, err := io.ReadFull(rand.Reader, buf[:16])
	if err != nil {
		return err
	}
	self.wg = new(sync.WaitGroup)
	self.wg.Add(self.N)
	for i := 0; i < self.N; i++ {
		go Client(self.Addr, buf[:], self.M, self.start, self.stop, self.resChan, self.wg)
	}
	return nil
}

func (self *BenchClient) collectResults() {
	if self.out == nil {
		self.out = os.Stdout
	}
	for r := range self.resChan {
		if r.err != nil {
			fmt.Fprintf(self.out, "Failed: %v\n", r.err)
		} else {
			fmt.Fprintf(self.out, "%v\n", r.d.Seconds())
		}
	}
}

func (self *BenchClient) Start() {
	go self.collectResults()
	close(self.start)
}

func (self *BenchClient) Wait() {
	self.wg.Wait()
}

var argvNrConn = flag.Int("n", 10, "number of concurrent connections")
var argvNrMsg = flag.Int("m", 10, "number of messages per connection")
var argvServAddr = flag.String("addr", "127.0.0.1:8080", "server address")
var argvOut = flag.String("o", "", "output file name")

func main() {
	flag.Parse()
	r := bufio.NewReader(os.Stdin)
	b := new(BenchClient)
	b.Addr = *argvServAddr
	b.N = *argvNrConn
	b.M = *argvNrMsg
	if len(*argvOut) > 0 {
		f, err := os.Create(*argvOut)
		if err != nil {
			fmt.Fprint(os.Stderr, "cannot create file: %v\n", err)
			return
		}
		b.out = f
	}

	fmt.Printf("Ready to start the connections? [Enter] ")
	r.ReadLine()
	b.Connect()

	fmt.Printf("Ready to start sending? [Enter] ")
	r.ReadLine()
	b.Start()

	b.Wait()
	fmt.Printf("Hit Enter to stop")
	r.ReadLine()
}
