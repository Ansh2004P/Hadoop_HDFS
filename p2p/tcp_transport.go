package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
)

// TCPPeer represents the remote node over a TCP established connection.
type TCPPeer struct {
	// conn is underlying connection of peer. Which in this case 
	// is a TCP connection.
	net.Conn

	// if we dial and retrieve a conn =>outbound ==true
	// if we accept and retrieve a conn =>inbound ==true or outbound == false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		outbound: outbound,
	}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Write(b)
	return err
}

func formatAddr(addr net.Addr) string {
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		if tcpAddr.IP.IsLoopback() {
			return fmt.Sprintf("127.0.0.1:%d", tcpAddr.Port)
		}
		fmt.Print(addr.String())
		// For other IPv6 addresses, use IPv4 if available
		if ipv4 := tcpAddr.IP.To4(); ipv4 != nil {
			return fmt.Sprintf("%s:%d", ipv4.String(), tcpAddr.Port)
		}
	}
	return addr.String()
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) TCPTransport {
	return TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}
}

func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// Consume implements the Transport interface, which will return a read-only channel
// for reading the incoming messages received from another peer in the network.
func (t *TCPTransport) Consume() <-chan RPC {
	// fmt.Printf("TCP: consume channel created\n");
	return t.rpcch
}

// Close implements the Transport interface 
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
}
 
func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}
	go t.startAcceptLoop()

	log.Println("TCP: listening on ", t.ListenAddr)
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}

		if err != nil {
			fmt.Printf("TCP: accept error: %s\n", err)
			continue
		}

		clientIP := formatAddr(conn.RemoteAddr())
		fmt.Printf("New client connected from: %s on %s\n", clientIP, t.ListenAddr)

		go t.handleConn(conn, false)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		fmt.Printf("Dropping peer connection: %s", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	if err := t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err := t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop
	rpc := RPC{}
	for {
		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			return
		}

		rpc.From = conn.RemoteAddr()
		t.rpcch <- rpc
		clientIP := formatAddr(conn.RemoteAddr())

		fmt.Printf("Message received from %s - Payload: %v\n", clientIP, rpc.Payload)
	}
}
