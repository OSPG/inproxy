package inproxy

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// Parse HTTP requests
// https://godoc.org/net/http#ReadRequest

// TODO: Add proper logging system

// ProxyServer is the struct containing the main elements required by the proxy
// to run.
type ProxyServer struct {
	listener net.Listener
	Ladress  string

	initialized bool
	Verbose     bool

	// Buffer where pending requests will wait to be processed, for example
	// while the user is editing one request.
	requestsBuffer chan net.Conn

	// Length of requestsBuffer
	requestsBufferLen int

	// True if the proxy is listening for connections, false if it isn't.
	running bool

	// True if the proxy is intercepting, false if it isn't.
	enabled bool

	requestHandler func(*http.Request, bufio.Reader) *http.Request
}

func (p *ProxyServer) Init() {
	p.requestsBufferLen = 100 // Arbitrary
	p.requestsBuffer = make(chan net.Conn, p.requestsBufferLen)
	p.initialized = true
}

func (p *ProxyServer) Serve() error {
	if !p.initialized {
		return errors.New("inproxy: Proxy not initialized")
	}

	fmt.Println("Serving")

	var err error
	if p.listener, err = net.Listen("tcp", p.Ladress); err != nil {
		return err
	}

	fmt.Println("Listener created")

	p.running = true
	p.enabled = true

	go p.handleConns()

	// Anonymous function accepting connections and passing the net.Conn objects
	// to ProxyServer.handleConn() through the ProxyServer.requestsBuffer chan
	go func() {
		for p.running {
			conn, err := p.listener.Accept()
			if err != nil {
				fmt.Println("ERROR: p.listener.Accept(): ", err)
			} else {
				// conn variable is sent to the channel only if it's not full
				fmt.Println("Accepting connection")
				select {
				case p.requestsBuffer <- conn:
				default:
					fmt.Println("ERROR: requests chan is full")
				}
			}
		}
	}()

	return nil
}

func (p *ProxyServer) handleConns() {
	// TODO: keep-alive connections
	var conn net.Conn
	for p.running {
		conn = <-p.requestsBuffer
		fmt.Println("New connection")
		reader := bufio.NewReader(conn)
		request, err := http.ReadRequest(reader)
		if err != nil {
			fmt.Println("ERROR: Can't parse request", err)
		} else {
			fmt.Println("SUCCESS: HTTP request parsed:")
			fmt.Println("\rMethod:", request.Method)
			fmt.Println("\rUrl:   ", request.URL.String())
			fmt.Println("\rProto: ", request.Proto)
			fmt.Println("\rHost:  ", request.Host)
			// ...

		}
		conn.Close()
	}
}

func NewProxy(laddr string, verbose bool) *ProxyServer {
	proxy := &ProxyServer{
		Ladress: laddr,
		Verbose: verbose}
	proxy.Init()
	return proxy
}
