package main

import (
    "fmt"
    "net"
    "os"
    "io"
    "log"
    "bufio"
    "strings"
    "regexp"
    "net/url"
    "github.com/oguzbilgic/socketio"
    "github.com/bitly/go-simplejson"
)

func connectToWeblights() {
    // Open a new client connection to the given socket.io server
    // Connect to the given channel on the socket.io server
    socket, err := socketio.Dial("http://weblights.ehrat.farm:80")

    if err != nil {
        panic(err)
    }

    for {
        // Receive socketio.Message from the server
        msg, err := socket.Receive()
        if err != nil {
            panic(err)
        }
        //dec := json.NewDecoder(strings.NewReader(msg.Data))
        //var data SioMsgData
        //dec.Decode(&data)
        //json.Unmarshal([]byte(msg.Data), &data)
        json := new(simplejson.Json)
        jerr := json.UnmarshalJSON([]byte(msg.Data))
        log.Println(jerr)
        name, _ := json.Get("name").String()
        if name == "dc_lights" {
            status, _ := json.Get("args").GetIndex(0).Bool()
            log.Printf("dc_lights: %t\n", status)
        }

        fmt.Printf("Type: %v, ID: '%s', Endpoint: '%s', Data: '%s' \n", msg.Type, msg.ID, msg.Endpoint, msg.Data)
    }
}

func main() {
//    connectToWeblights()

    port := "554"

    ln, err := net.Listen("tcp", ":" + port)
    if err != nil {
        log.Printf("Could not listen on port %d. %s\n", port, err.Error())
        os.Exit(1)
    }
    log.Printf("Listening on %s\n", ln.Addr().String())
    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Println("Error accepting connection: ", err.Error())
        } else {
  	    log.Printf("New Connection from %s\n", conn.RemoteAddr().String())
            go handleFrontend(conn)
        }
    }

}

func handleFrontend(conn net.Conn) {
  clog := log.New(os.Stdout, fmt.Sprintf("[%s] ", conn.RemoteAddr().String()), log.LstdFlags)
  
  buf_reader := bufio.NewReader(conn)

  // Get the first line, which should in contain the RTSP Method request
  request_line, err := buf_reader.ReadString('\n')
  if err != nil {
    clog.Println("Error reading request: ", err.Error())
    clog.Println("Closing")
    conn.Close()
    return
  }

  request_regex := regexp.MustCompile(`([A-Z]*) (.*) RTSP/(.*)\r\n`)
  request_params := request_regex.FindStringSubmatch(request_line)
  if len(request_params) != 4 {
    clog.Printf("Could not understand request: %s", request_line)
    clog.Println("Closing")
    conn.Close()
    return
  }

  request_str := request_params[0]
  method := request_params[1]
  uri := request_params[2]
  rtsp_version := request_params[3]

  clog.Printf("Request: %s", request_str) // Already has /n

  if method != "OPTIONS" {
    clog.Printf("Received method %s instead of OPTIONS\n", request_params[0])
    clog.Println("Closing")
    conn.Close()
    return
  }

  if rtsp_version != "1.0" {
    clog.Printf("Received RTSP version %s instead of 1.0\n", request_params[3])
    clog.Println("Closing")
    conn.Close()
    return
  }

  // Parse the URL and ensure there are no errors.
  url, err := url.Parse(uri)
  if err != nil {
    clog.Printf("Could not parse request uri: %s. %s\n", uri, err.Error())
    clog.Println("Closing")
    conn.Close()
    return
  }

  hostport := strings.Split(url.Host, ":")
  forward_host := hostport[0] + ":554"
  clog.Println("Forwarding to host: ", forward_host)

  forward_conn, err := net.Dial("tcp", forward_host)
  if err != nil {
	clog.Printf("Could not connect to forward host %s. %s\n", forward_host, err.Error())
	clog.Println("Closing")
	conn.Close()
	return
  }

  _, err = forward_conn.Write([]byte(request_line))
  if err != nil {
        clog.Printf("Could not write forward request: %s\n", err)
	clog.Println("Closing")
	conn.Close()
        forward_conn.Close()
	return
  }


  reverse_chan := make(chan int64)
  go connCopy(forward_conn, conn, clog, reverse_chan)

  forward_bytes, err := buf_reader.WriteTo(forward_conn)
  if err != nil {
    clog.Printf("Forward copy error: %s\n", err)
  }
	clog.Printf("Wrote %d bytes to forward connection\n", forward_bytes + int64(len(request_line)))


        if err := forward_conn.Close(); err != nil {
                clog.Printf("Forward conn close error: %s", err)
        }
        if err := conn.Close(); err != nil {
                clog.Printf("Reverse conn close error: %s", err)
        }
  reverse_bytes := <-reverse_chan
  clog.Printf("Forward thread got reverse_bytes: %d\n", reverse_bytes)

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
                clog.Printf("Reverse copy error: %s", err)
        }

        if err := src.Close(); err != nil {
                clog.Printf("Reverse src close error: %s", err)
        }
        if err := dst.Close(); err != nil {
                clog.Printf("Reverse dst close error: %s", err)
        }
        
	clog.Printf("Wrote %d bytes to reverse connection\n", reverse_bytes)
	reverse_chan <- reverse_bytes
}

