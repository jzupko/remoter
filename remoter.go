package main

import (
	"github.com/namsral/flag"
	"golang.design/x/clipboard"

	"bufio"
	"context"
	"encoding/gob"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	DefaultListenPort = ":30359"
)

var (
	errorLog = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime)
	infoLog  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
)

func onError(err error) error {
	if err != nil {
		errorLog.Println(err)
	}
	return err
}

type entry struct {
	Format clipboard.Format
	Data   []byte
}

func recvCommon(conn net.Conn) {
	for {
		var entry entry
		r := bufio.NewReader(conn)
		dec := gob.NewDecoder(r)
		err := dec.Decode(&entry)
		if err != nil {
			if err == io.EOF {
				return
			} else {
				onError(err)
			}
			continue
		}

		if len(entry.Data) == 0 {
			continue
		}

		clipboard.Write(entry.Format, entry.Data)
	}
}

func sendCommon(conn net.Conn, entry entry) {
	if conn == nil {
		return
	}

	w := bufio.NewWriter(conn)
	enc := gob.NewEncoder(w)
	err := enc.Encode(entry)
	if onError(err) == nil {
		onError(w.Flush())
	}
}

type server struct {
	addr  string
	mutex sync.RWMutex
	conns []net.Conn
}

func (s *server) remove(conn net.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i := 0; i < len(s.conns); i++ {
		if s.conns[i] == conn {
			s.conns = append(s.conns[:i], s.conns[i+1:]...)
			return
		}
	}
}

func (s *server) store(conn net.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.conns = append(s.conns, conn)
}

func (s *server) perClientBody(conn net.Conn) {
	if conn == nil {
		return
	}
	defer conn.Close()

	s.store(conn)
	recvCommon(conn)
	s.remove(conn)
}

func (s *server) send(fmt clipboard.Format, data []byte) {
	entry := entry{
		Format: fmt,
		Data:   data,
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, conn := range s.conns {
		sendCommon(conn, entry)
	}
}

func (s *server) monitor() {
	textChan := clipboard.Watch(context.Background(), clipboard.FmtText)
	imgChan := clipboard.Watch(context.Background(), clipboard.FmtImage)
	for {
		select {
		case textData := <-textChan:
			s.send(clipboard.FmtText, textData)
		case imgData := <-imgChan:
			s.send(clipboard.FmtImage, imgData)
		}
	}
}

func (s *server) main() {
	listener, err := net.Listen("tcp4", s.addr)
	if onError(err) != nil {
		os.Exit(1)
	}
	defer listener.Close()

	infoLog.Println("Listening on:", s.addr)
	setStatus("Listening on: %s", s.addr)

	go s.monitor()

	for {
		conn, err := listener.Accept()
		if onError(err) != nil {
			continue
		}

		go s.perClientBody(conn)
	}
}

type client struct {
	addr  string
	mutex sync.RWMutex
	conn  net.Conn
}

func (c *client) send(fmt clipboard.Format, data []byte) {
	entry := entry{
		Format: fmt,
		Data:   data,
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.conn != nil {
		sendCommon(c.conn, entry)
	}
}

func (c *client) monitor() {
	textChan := clipboard.Watch(context.Background(), clipboard.FmtText)
	imgChan := clipboard.Watch(context.Background(), clipboard.FmtImage)
	for {
		select {
		case textData := <-textChan:
			c.send(clipboard.FmtText, textData)
		case imgData := <-imgChan:
			c.send(clipboard.FmtImage, imgData)
		}
	}
}

func (c *client) receive() {
	if c.conn == nil {
		return
	}

	recvCommon(c.conn)
}

func (c *client) connect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var err error
	c.conn, err = net.Dial("tcp4", c.addr)
	if err != nil || c.conn == nil {
		c.conn = nil
		return
	}

	infoLog.Println("Connected to:", c.addr)
	setStatus("Connected to: %s", c.addr)
}

func (c *client) disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	setStatus("Disconnected")

	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = nil
}

func (c *client) main() {
	go c.monitor()

	for {
		c.connect()
		c.receive()
		c.disconnect()
	}
}

func mainBody() {
	flags := flag.NewFlagSetWithEnvPrefix("remoter command-line flags", "REMOTER", flag.ExitOnError)
	flagConnect := flags.String("connect", "", "<host>:<port> to connect to in client mode.")
	flagPort := flags.String("port", DefaultListenPort, "<host>:<port> of the host in server mode.")
	flags.Parse(os.Args[1:])

	if onError(clipboard.Init()) != nil {
		os.Exit(1)
	}

	if "" == *flagConnect {
		s := server{
			addr: *flagPort,
		}
		s.main()
	} else {
		// Allow no port for client - in this case, fall back to default port.
		addr := *flagConnect
		if !strings.Contains(addr, ":") {
			addr += DefaultListenPort
		}

		c := client{
			addr: addr,
		}
		c.main()
	}
}
