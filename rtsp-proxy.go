package main


import (
    "fmt"
    "net"
    "os"
    "io"
    "log"
    "bufio"
    "github.com/oguzbilgic/socketio"
    "github.com/bitly/go-simplejson"
)

type SioMsgData struct {
    name string
    args []bool
}

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
    connectToWeblights()

    ln, err := net.Listen("tcp", ":5554")
    if err != nil {
        // handle error
        fmt.Println("Error Listening:", err.Error())
        os.Exit(1)
    }
    for {
        conn, err := ln.Accept()
        if err != nil {
            // handle error
        } else {
            go handleFrontend(conn)
        }
    }

}


func handleFrontend(conn net.Conn) {
  // Make a buffer to hold incoming data.
  //buf := make([]byte, 1024)
  buf_reader := bufio.NewReader(conn)
  buf_writer := bufio.NewWriter(conn)
  request, err := buf_reader.ReadString('\n')
  // Read the incoming connection into the buffer.
  //_, err := conn.Read(buf)
  if err != nil {
    fmt.Println("Error reading:", err.Error())
  }
  // Send a response back to person contacting us.
  //conn.Write([]byte("Message received."))
  // Close the connection when you're done with it.
  buf_writer.WriteString("Got Header: " + request)
  buf_writer.Flush()
  conn.Close()
}

// This does the actual data transfer.
// The broker only closes the Read side.
func connCopy(dst, src net.Conn, srcClosed chan struct{}) {
        // We can handle errors in a finer-grained manner by inlining io.Copy (it's
        // simple, and we drop the ReaderFrom or WriterTo checks for
        // net.Conn->net.Conn transfers, which aren't needed). This would also let
        // us adjust buffersize.
        _, err := io.Copy(dst, src)
 
        if err != nil {
                log.Printf("Copy error: %s", err)
        }
        if err := src.Close(); err != nil {
                log.Printf("Close error: %s", err)
        }
        srcClosed <- struct{}{}
}

