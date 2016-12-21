package inproxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// parseRequest accept a request as bufio.Reader and returns the HTTP request as
// bytes.Buffer (the raw request) or as http.Request (the parsed request)
func parseRequest(reader *bufio.Reader) (*bytes.Buffer, *http.Request, error) {
	rawReq := new(bytes.Buffer)

	// Get first line of the request: GET http://example.com HTTP/1.1
	// TODO: \r\n instead of \n
	s, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, nil, errors.New("inproxy: error reading request line")
	}

	// fmt.Println("DEBUG: request line: ", string(s))

	// Split the line to parse the method, URL, and protocol version
	// requestLine[0] == Method
	// requestLine[1] == URL
	// requestLine[2] == Version
	requestLine := bytes.Split(s, []byte(" "))
	if len(requestLine) != 3 {
		return nil, nil, errors.New("inproxy: invalid request line")
	}

	rawReq.Write(s)

	switch string(requestLine[0]) {
	case "GET":
		if err = parseGet(reader, rawReq); err != nil {
			return nil, nil, errors.New("inproxy: error parsing GET request")
		}
		break
	default:
		fmt.Println("INFO: Method ", string(requestLine[0]), " not implemented yet")
		break
	}

	// If we don't use bytes.NewBuffer() we'd lose the content of rawReq
	// idk if it's the best solution but for now it just works
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(rawReq.Bytes())))
	if err != nil {
		return nil, nil, errors.New("inproxy: can't parse request")
	}

	return rawReq, req, nil
}

// parseGet gets a reader (bufio.Reader) and reads line by line until it receive
// a line with just CRLF ('\r\n').
// RFC 7230 Section 3
func parseGet(reader *bufio.Reader, rawReq *bytes.Buffer) error {
	for crlf := false; !crlf; {
		// TODO: \r\n instead of \n
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return errors.New("inproxy: parseGet: Error reading line")
			}
		}

		if bytes.Equal(line, []byte("\r\n")) {
			crlf = true
		}

		// fmt.Println("DEBUG: Read line (string)", string(line))
		// fmt.Println("DEBUG: Read line ([]byte)", line)

		rawReq.Write(line)
	}
	return nil
}
