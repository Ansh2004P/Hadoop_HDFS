package main

import (
	"log"
	"time"

	"github.com/Ansh2004P/hdfs/p2p"
)

// ListenAddr    string
// 	HandshakeFunc HandshakeFunc
// 	Decoder       Decoder
// 	OnPeer        func(Peer) error

func main() {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO: OnPeer func
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         *tcpTransport,
	}

	s := NewFileServer(fileServerOpts)
	
		go func() {
			time.Sleep(time.Second*3)
			s.Stop()
		}()
	
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
