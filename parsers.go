package inproxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// parseRequest accepts a request as bufio.Reader and returns the HTTP request
// as bytes.Buffer (the raw request) and as http.Request (the parsed request).
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
		headers, err := parseHeaders(reader, rawReq)
		if err != nil {
			fmt.Println("DEBUG: GET parser error", err)
			return nil, nil, errors.New("inproxy: error parsing GET request")
		}
		fmt.Println("DEBUG: Parsed headers")
		for k, v := range *headers {
			fmt.Println("DEBUG: ", k, " ", v)
		}
	case "HEAD":
		fallthrough
	case "POST":
		fallthrough
	case "PUT":
		fallthrough
	case "DELETE":
		fallthrough
	case "CONNECT":
		fallthrough
	case "OPTIONS":
		fallthrough
	case "TRACE":
		return nil, nil, errors.New("inproxy: HTTP method not implemented yet")
	default:
		return nil, nil, errors.New("inproxy: invalid method in HTTP request")
	}

	// If we don't use bytes.NewBuffer() we'd lose the content of rawReq
	// idk if it's the best solution but for now it just works
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(rawReq.Bytes())))
	if err != nil {
		return nil, nil, errors.New("inproxy: can't parse request")
	}

	return rawReq, req, nil
}

// parseHeaders gets a reader (bufio.Reader) and reads line by line until it
// receive a blank line ('\r\n'). It updates rawReq with the headers
// data and returns the parsed headers as *http.Header
func parseHeaders(reader *bufio.Reader, rawReq *bytes.Buffer) (*http.Header, error) {
	headers := http.Header{}
	for crlf := false; !crlf; {
		// TODO: \r\n instead of \n
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return nil, errors.New("inproxy: error reading line")
			}
		}

		// Check if it's a blank line
		if bytes.Equal(line, []byte("\r\n")) {
			crlf = true
		} else {
			// Header parsing stuff
			// RFC 7230 Section 3.2

			// Remove spaces between header field-name and colon, as specified
			// in RFC 7230 Section 3.2.4
			// Host : example.com --> Host: example.com
			var newLine bytes.Buffer
			afterColon := false
			for i := 0; i < len(line); i++ {
				if (line[i] != ' ' && !afterColon) || afterColon {
					newLine.WriteByte(line[i])
				}
				if line[i] == ':' {
					afterColon = true
				}
			}

			line = newLine.Bytes()

			var (
				key   string
				value string
			)

			index := bytes.IndexByte(line, ':')

			if index < 0 {
				fmt.Println("DEBUG: line: ")
				fmt.Println(line)
				fmt.Println(string(line))
				return nil, errors.New("inproxy: malformed header line")
			}

			key = string(line[:index])
			if key == "" {
				// field-name can't be empty, we don't raise an error, just
				// ignore this header.
				fmt.Println("INFO: Empty field-name in header")
				continue
			}

			// Skip colon
			index++

			// Ignore optional whitespaces or tabs before the field-value
			// field-name ":" OWS field-value OWS
			for index < len(line) && (line[index] == ' ' || line[index] == '\t') {
				index++
			}

			// tmpValue contains the value without the optional initial
			// whitespaces, but it may contain whitespaces at the end.
			tmpValue := line[index:]

			// Ignore optional whitespaces, CRLF or tabs after the field-value
			reverseIndex := len(tmpValue)
			for reverseIndex > 0 && (tmpValue[reverseIndex-1] == '\n' ||
				tmpValue[reverseIndex-1] == '\r' ||
				tmpValue[reverseIndex-1] == ' ' ||
				tmpValue[reverseIndex-1] == '\t') {

				reverseIndex--
			}
			value = string(tmpValue[:reverseIndex])

			headers.Add(key, value)
		}

		// fmt.Println("DEBUG: Read line (string)", string(line))
		// fmt.Println("DEBUG: Read line ([]byte)", line)

		rawReq.Write(line)
	}
	return &headers, nil
}
