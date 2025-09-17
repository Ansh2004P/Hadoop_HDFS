package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/Ansh2004P/hdfs/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	BootStrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store     *Store
	quitch    chan struct{}
	Transport p2p.TCPTransport
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

type Payload struct {
	Key  string
	Data []byte
}

func (s *FileServer) broadcast(p *Payload) error {
	peers := []io.Writer{}

	s.peerLock.Lock()
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	s.peerLock.Unlock()

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(p)
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key string
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	// FLOW:
	// 1. Store file to disk
	// 2. Broadcast this file to all peers in the network

	log.Printf("Storing data with key: %s", key)

	// Read the data into buffer first
	buf := new(bytes.Buffer)
	msg := Message{
		Payload: MessageStoreFile{Key: key},
	}

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}
	// time.Sleep(time.Second * 3)

	// payload := []byte("THIS LARGE FILE")

	// for _, peer := range s.peers {
	// 	if err := peer.Send(payload); err != nil {
	// 		return err
	// 	}
	// }

	return nil

	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)

	// if err := s.store.Write(key, tee); err != nil {
	// 	fmt.Println("Error storing data:", err)
	// 	return err
	// }
	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }
	// fmt.Println(p.Data)
	// return s.broadcast(&Message{
	// 	From:    "TODO",
	// 	Payload: p,
	// })
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	log.Printf("Connected with peer %s", p.RemoteAddr())

	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to user quit action")
		s.Transport.Close()
		log.Println("File server transport closed")
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Printf("ERROR: Failed to decode message: %v\n", err)
			}

			fmt.Printf("%+v\n", msg.Payload)

			peer, ok := s.peers[rpc.From]
			if !ok {
				panic("Peer not found in peers map")
			}

			b := make([]byte, 1000)
			if _, err := peer.Read(b); err != nil {
				panic(err)
			}

			fmt.Printf("%s\n", string(b))
			peer.(*p2p.TCPPeer).Wg.Done()
			// if err := s.handleMessage(&m); err != nil {
			// 	log.Println(err)
			// }

		case <-s.quitch:
			return
		}
	}
}

// func (s *FileServer) handleMessage(msg *Message) error {
// 	switch v := msg.Payload.(type) {
// 	case *DataMessage:
// 		fmt.Printf("Received Data %v\n", v)
// 	}
// 	return nil
// }

func (s *FileServer) bootStrapNetwork() error {
	for _, addr := range s.BootStrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("Dial error: ", err)
			}
		}(addr)
	}
	return nil
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.bootStrapNetwork()

	s.loop()
	return nil
}
