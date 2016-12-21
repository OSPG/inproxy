package inproxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// TODO: Add proper logging system

type RequestHandler func(*http.Request, *bytes.Buffer)

// ProxyServer is the struct containing the main elements required by the proxy
// to run.
type ProxyServer struct {
	// Adress and port to bind,
	// examples: ":8080", "127.0.0.1:8080", "0.0.0.0:8080".
	Ladress string
	Verbose bool

	listener net.Listener

	// True if the Init() method have been called, false if not
	initialized bool

	// Buffer where pending requests will wait to be processed, for example
	// while the user is editing one request.
	requestsBuffer chan net.Conn

	// Length of requestsBuffer
	requestsBufferLen int

	// True if the proxy is listening for connections, false if it isn't.
	running bool

	// True if the proxy is intercepting requests , false if it isn't.
	requestsEnabled bool

	// True if the proxy is intercepting responses from the server
	responseEnabled bool

	// Function called with every incoming valid HTTP request.
	// Args: Parsed HTTP request (*http.Request), and raw HTTP request
	// (*bytes.Buffer). The modified raw HTTP request will be sent to the server
	// any change at the parsed request will be ignored.
	requestHandler RequestHandler
}

// Init initializes the main parts of ProxyServer
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

// handleConns accept connections from the ProxyServer.requestsBuffer chan while
// the server is running.
// TODO: keep-alive connections
func (p *ProxyServer) handleConns() {
	var (
		conn net.Conn
	)
	for p.running {
		conn = <-p.requestsBuffer
		reader := bufio.NewReader(conn)
		fmt.Println("INFO: New connection")

		rawReq, request, err := parseRequest(reader)
		if err != nil {
			fmt.Println("ERROR: Error reading the request", err)
			conn.Close()
			continue
		}

		fmt.Println("RAW REQUEST:")
		fmt.Println(rawReq.String())

		fmt.Println("PARSED REQUEST:")
		fmt.Println("\tMethod:", request.Method)
		fmt.Println("\tUrl:   ", request.URL.String())
		fmt.Println("\tProto: ", request.Proto)
		fmt.Println("\tHost:  ", request.Host)
		// ...

		conn.Close()
	}
}

func (p *ProxyServer) SetRequestsHandler(handler RequestHandler) {
	p.requestHandler = handler
	p.requestsEnabled = true
}

func NewProxy(laddr string, verbose bool) *ProxyServer {
	proxy := &ProxyServer{
		Ladress: laddr,
		Verbose: verbose}
	proxy.Init()
	return proxy
}
