package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var (
	listen            = flag.String("listen", ":554", "Listen host:port")
	allowedHostRegexp = flag.String("allowed-host-regexp", "", "Hosts which to allow proxying to")
)

func main() {
	flag.Parse()

	var hostRegexp *regexp.Regexp
	if *allowedHostRegexp != "" {
		var err error
		hostRegexp, err = regexp.Compile(*allowedHostRegexp)
		if err != nil {
			log.Printf("Invalid allowed-host-regexp: %v\n", err)
			os.Exit(1)
		}
	}

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Printf("Could not listen on %d. %s\n", *listen, err.Error())
		os.Exit(1)
	}
	log.Printf("Listening on %s\n", ln.Addr().String())
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection: ", err.Error())
		} else {
			log.Printf("New Connection from %s\n", conn.RemoteAddr().String())
			go handleFrontend(conn, hostRegexp)
		}
	}

}

func handleFrontend(conn net.Conn, hostRegexp *regexp.Regexp) {
	clog := log.New(os.Stdout, fmt.Sprintf("[%s] ", conn.RemoteAddr().String()), log.LstdFlags)

	buf_reader := bufio.NewReader(conn)

	// Get the first line, which should in contain the RTSP Method request
	request_line, err := buf_reader.ReadString('\n')
	if err != nil {
		clog.Println("Error reading request: ", err.Error())
		conn.Close()
		return
	}

	request_regex := regexp.MustCompile(`([A-Z]*) (.*) RTSP/(.*)\r\n`)
	request_params := request_regex.FindStringSubmatch(request_line)
	if len(request_params) != 4 {
		clog.Printf("(Closing) Could not understand request: %s", request_line)
		conn.Close()
		return
	}

	//request_str := request_params[0]
	method := request_params[1]
	uri := request_params[2]
	rtsp_version := request_params[3]

	//clog.Printf("Request: %s", request_str) // Already has /n

	if method != "OPTIONS" {
		clog.Printf("(Closing) Received method %s instead of OPTIONS\n", request_params[0])
		conn.Close()
		return
	}

	if rtsp_version != "1.0" {
		clog.Printf("(Closing) Received RTSP version %s instead of 1.0\n", request_params[3])
		conn.Close()
		return
	}

	// Parse the URL and ensure there are no errors.
	url, err := url.Parse(uri)
	if err != nil {
		clog.Printf("(Closing) Could not parse request uri: %s. %s\n", uri, err.Error())
		conn.Close()
		return
	}

	hostport := strings.Split(url.Host, ":")
	host := hostport[0]

	if host == "" {
		clog.Printf("(Closing) No host given")
		conn.Close()
		return
	}

	if hostRegexp != nil && hostRegexp.FindString(host) != host {
		clog.Printf("(Closing) Unallowed host: %v", host)
		conn.Close()
		return
	}

	forward_host := host + ":554"
	clog.Println("Forwarding to ", forward_host)

	forward_conn, err := net.Dial("tcp", forward_host)
	if err != nil {
		clog.Printf("(Closing) Could not connect to forward host %s. %s\n",
			forward_host, err.Error())
		conn.Close()
		return
	}

	// Write the request line forward
	_, err = forward_conn.Write([]byte(request_line))
	if err != nil {
		clog.Printf("(Closing) Could not write forward request: %s\n", err)
		conn.Close()
		forward_conn.Close()
		return
	}

	// Forward server -> client
	reverse_chan := make(chan int64)
	go connCopy(forward_conn, conn, clog, reverse_chan)

	// Forward client -> server
	forward_bytes, err := buf_reader.WriteTo(forward_conn)
	if err != nil {
		clog.Printf("Client closed connection: %s\n", err)
	}
	//clog.Printf("Wrote %d bytes to forward connection\n", forward_bytes+int64(len(request_line)))

	// close all connections discarding errors
	// this is force the other thread to close
	_ = forward_conn.Close()
	_ = conn.Close()

	// get reverse bytes from oother thread
	reverse_bytes := <-reverse_chan
	clog.Printf("Wrote %d bytes to client and %d bytes to server\n",
		reverse_bytes, forward_bytes)
}

// This does the actual data transfer.
// The broker only closes the Read side.
func connCopy(src, dst net.Conn, clog *log.Logger, reverse_chan chan int64) {
	// We can handle errors in a finer-grained manner by inlining io.Copy (it's
	// simple, and we drop the ReaderFrom or WriterTo checks for
	// net.Conn->net.Conn transfers, which aren't needed). This would also let
	// us adjust buffersize.
	reverse_bytes, err := io.Copy(dst, src)
	if err != nil {
		clog.Printf("Server closed connection: %s", err)
	}

	// close all connections discarding errors
	// this is force the other thread to close
	src.Close()
	dst.Close()

	//clog.Printf("Wrote %d bytes to reverse connection\n", reverse_bytes)
	reverse_chan <- reverse_bytes
}
