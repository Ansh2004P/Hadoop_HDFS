package main

import (
	"bytes"
	"log"
	"strings"
	"time"

	"github.com/Ansh2004P/hdfs/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	// First create the server without transport
	fileServerOpts := FileServerOpts{
		StorageRoot:       strings.ReplaceAll(listenAddr, ":", "") + "_network",
		PathTransformFunc: CASPathTransformFunc,
		BootStrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)

	// Now create transport with OnPeer callback
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        s.OnPeer, // Set OnPeer in the options
	}

	log.Printf("Creating HDFS server on %s", listenAddr)
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	// Set the transport in the server
	s.Transport = *tcpTransport

	return s
}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()
	time.Sleep(1 * time.Second)

	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(1 * time.Second)

	data := bytes.NewReader([]byte("My big data file here!"))

	s2.StoreData("myprivatedata", data)

	select {}
}
