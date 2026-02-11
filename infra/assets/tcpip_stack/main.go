package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ethernet"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/icmp"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/rawsock"
)

// Stack represents the TCP/IP stack
type Stack struct {
	sock   *rawsock.Socket
	iface  string
	srcMAC net.HardwareAddr
	srcIP  net.IP
}

// NewStack creates a new TCP/IP stack
func NewStack(iface string) (*Stack, error) {
	sock, err := rawsock.Open(iface)
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

	// Get interface MAC and IP
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get interface: %w", err)
	}

	addrs, err := ifaceObj.Addrs()
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get addresses: %w", err)
	}

	var srcIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			// Skip localhost
			if !ipnet.IP.IsLoopback() {
				srcIP = ipnet.IP
				break
			}
		}
	}

	if srcIP == nil {
		sock.Close()
		return nil, fmt.Errorf("no IPv4 address found for interface %s", iface)
	}

	return &Stack{
		sock:   sock,
		iface:  iface,
		srcMAC: ifaceObj.HardwareAddr,
		srcIP:  srcIP,
	}, nil
}

// Run starts the packet processing loop
func (s *Stack) Run(ctx context.Context) error {
	buf := make([]byte, 65536)
	log.Printf("Listening on %s (%s, MAC: %s)", s.iface, s.srcIP, s.srcMAC)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, err := s.sock.Recv(buf)
			if err != nil {
				continue
			}

			packetData := make([]byte, n)
			copy(packetData, buf[:n])

			if err := s.handlePacket(packetData); err != nil {
				log.Printf("handle packet: %v", err)
			}
		}
	}
}

// handlePacket processes a single packet
func (s *Stack) handlePacket(data []byte) error {
	// Parse Ethernet frame
	frame, err := ethernet.Parse(data)
	if err != nil {
		return err
	}

	// Filter: Only process packets for our MAC or broadcast/multicast
	if !s.shouldProcessFrame(frame) {
		return nil
	}

	// Handle IPv4
	if frame.EtherType != ethernet.TypeIPv4 {
		return nil
	}

	pkt, err := ipv4.Parse(frame.Payload)
	if err != nil {
		return err
	}

	// Log non-ICMP packets for debugging
	if pkt.Protocol != ipv4.ProtocolICMP {
		log.Printf("Got %s -> %s", pkt.SrcIP, pkt.DstIP)
	}

	// Only handle ICMP for this implementation
	if pkt.Protocol != ipv4.ProtocolICMP {
		return nil
	}

	msg, err := icmp.Parse(pkt.Payload)
	if err != nil {
		return err
	}

	// Respond to Echo Request (ping)
	if msg.IsEchoRequest() && pkt.DstIP.Equal(s.srcIP) {
		log.Printf("Ping from %s: ID=%d Seq=%d", pkt.SrcIP, msg.ID, msg.Seq)
		return s.sendEchoReply(pkt, msg, frame.SrcMAC)
	}

	return nil
}

// shouldProcessFrame checks if we should process this frame
func (s *Stack) shouldProcessFrame(frame *ethernet.Frame) bool {
	// Process if addressed to us
	if frame.DstMAC.String() == s.srcMAC.String() {
		return true
	}
	// Process if broadcast
	if ethernet.IsBroadcast(frame.DstMAC) {
		return true
	}
	// Process if multicast
	if ethernet.IsMulticast(frame.DstMAC) {
		return true
	}
	return false
}

// sendEchoReply sends an ICMP Echo Reply
func (s *Stack) sendEchoReply(reqPkt *ipv4.Packet, reqMsg *icmp.Message, dstMAC net.HardwareAddr) error {
	// Build ICMP Echo Reply
	reply := icmp.NewEchoReply(reqMsg.ID, reqMsg.Seq, reqMsg.Data)
	icmpData := reply.Marshal()

	// Build IP packet
	ipPkt := &ipv4.Packet{
		Version:     4,
		IHL:         20,
		TOS:         0,
		TTL:         64,
		Protocol:    ipv4.ProtocolICMP,
		SrcIP:       s.srcIP,
		DstIP:       reqPkt.SrcIP,
		Payload:     icmpData,
	}

	ipData := ipPkt.Marshal()

	// Build Ethernet frame
	frame := &ethernet.Frame{
		DstMAC:    dstMAC,
		SrcMAC:    s.srcMAC,
		EtherType: ethernet.TypeIPv4,
		Payload:   ipData,
	}

	frameData := frame.Marshal()

	// Send
	_, err := s.sock.Send(frameData)
	return err
}

// Close closes the stack
func (s *Stack) Close() error {
	return s.sock.Close()
}

func main() {
	iface := flag.String("iface", "", "Network interface to bind (required)")
	flag.Parse()

	if *iface == "" {
		fmt.Println("TCP/IP Protocol Stack Workshop")
		fmt.Println("Usage: sudo go run main.go -iface <interface>")
		fmt.Println("")
		fmt.Println("Example:")
		fmt.Println("  sudo go run main.go -iface eth0")
		fmt.Println("")
		fmt.Println("Available interfaces:")
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, i := range ifaces {
				addrs, _ := i.Addrs()
				addrStr := ""
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
						addrStr = ipnet.IP.String()
						break
					}
				}
				if addrStr != "" {
					fmt.Printf("  %s: %s (%s)\n", i.Name, addrStr, i.HardwareAddr)
				}
			}
		}
		os.Exit(1)
	}

	stack, err := NewStack(*iface)
	if err != nil {
		log.Fatalf("NewStack: %v", err)
	}
	defer stack.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("TCP/IP stack running on %s (%s)", *iface, stack.srcIP)
	log.Printf("Try: ping %s", stack.srcIP)

	if err := stack.Run(ctx); err != nil {
		log.Fatalf("Run: %v", err)
	}

	log.Println("TCP/IP stack shut down gracefully")
}
