package inproxy

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// Log levels
const (
	PanicLevel log.Level = iota
	ErrorLevel
	InfoLevel
	DebugLevel
)

type RequestHandler func(*http.Request, *bytes.Buffer)

// ProxyServer is the struct containing the main elements required by the proxy
// to run.
type ProxyServer struct {
	// Adress and port to bind,
	// examples: ":8080", "127.0.0.1:8080", "0.0.0.0:8080".
	Ladress string

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

	log.Info("Serving")

	var err error
	if p.listener, err = net.Listen("tcp", p.Ladress); err != nil {
		return err
	}

	log.Info("Listener created")

	p.running = true

	go p.handleConns()

	// Anonymous function accepting connections and passing the net.Conn objects
	// to ProxyServer.handleConn() through the ProxyServer.requestsBuffer chan
	go func() {
		for p.running {
			conn, err := p.listener.Accept()
			if err != nil {
				log.Error("p.listener.Accept(): ", err)
			} else {
				// conn variable is sent to the channel only if it's not full
				select {
				case p.requestsBuffer <- conn:
				default:
					log.Error("requests chan is full")
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

		log.Info("New connection")

		// rawReq, request, err := parseRequest(reader)
		rawReq, _, err := parseRequest(reader)
		if err != nil {
			log.Error("Could not read the request: ", err)
			conn.Close()
			continue
		}

		log.Debugf("%s", rawReq.Bytes())

		conn.Close()
	}
}

func (p *ProxyServer) SetRequestsHandler(handler RequestHandler) {
	p.requestHandler = handler
	p.requestsEnabled = true
}

func NewProxy(laddr string, loglevel log.Level) *ProxyServer {
	if loglevel == DebugLevel {
		log.SetLevel(log.DebugLevel)
	} else if loglevel == InfoLevel {
		log.SetLevel(log.InfoLevel)
	} else if loglevel == ErrorLevel {
		log.SetLevel(log.ErrorLevel)
	} else if loglevel == PanicLevel {
		log.SetLevel(log.PanicLevel)
	}

	proxy := &ProxyServer{
		Ladress: laddr,
	}

	proxy.Init()
	return proxy
}
