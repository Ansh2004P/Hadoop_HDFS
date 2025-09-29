package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/Ansh2004P/hdfs/p2p"
)

// sanitizeRoot converts a listen address like ":5000" or "127.0.0.1:5000" into a
// filesystem-friendly folder name (e.g. "5000_network" or "127.0.0.1_5000_network").
// Windows does not allow ':' in path components, so we strip/replace it.
func sanitizeRoot(listenAddr string) string {
	addr := strings.TrimSpace(listenAddr)
	// Remove leading colon (common pattern like ":3000")
	addr = strings.TrimPrefix(addr, ":")
	// Replace remaining colons (e.g. in "127.0.0.1:3000") with underscore.
	addr = strings.ReplaceAll(addr, ":", "_")
	if addr == "" {
		addr = "node"
	}
	return addr + "_network"
}

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcptransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcptransportOpts)

	fileServerOpts := FileServerOpts{
		EncKey:            newEncryptionKey(),
		StorageRoot:       sanitizeRoot(listenAddr),
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":7000", "")
	s3 := makeServer(":5000", ":3000", ":7000")

	go func() { log.Fatal(s1.Start()) }()
	time.Sleep(500 * time.Millisecond)
	go func() { log.Fatal(s2.Start()) }()

	time.Sleep(2 * time.Second)

	go s3.Start()
	time.Sleep(2 * time.Second)

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("picture_%d.png", i)
		data := bytes.NewReader([]byte("my big data file here!"))
		s3.Store(key, data)
		time.Sleep(500 * time.Millisecond) // Allow file operations to complete

		if err := s3.store.Delete(s3.ID, key); err != nil {
			log.Fatal(err)
		}

		r, err := s3.Get(key)
		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		// Close the file handle if it's a ReadCloser
		if rc, ok := r.(io.ReadCloser); ok {
			rc.Close()
		}

		fmt.Println(string(b))
	}
}
