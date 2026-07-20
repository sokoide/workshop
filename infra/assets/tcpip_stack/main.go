//go:build linux

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ethernet"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/icmp"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/rawsock"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/udp"
)

// Stack represents the TCP/IP stack
type Stack struct {
	sock   *rawsock.Socket
	iface  string
	srcMAC net.HardwareAddr
	srcIP  net.IP
}

// NewStack creates a new TCP/IP stack
func NewStack(iface string, srcIP net.IP) (*Stack, error) {
	sock, err := rawsock.Open(iface)
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

	// Get the interface MAC. The IP address is deliberately supplied by the
	// caller, so the Linux kernel does not also own and answer for it.
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get interface: %w", err)
	}

	if srcIP = srcIP.To4(); srcIP == nil {
		sock.Close()
		return nil, fmt.Errorf("source IP must be IPv4")
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
	var closeOnce sync.Once
	go func() {
		<-ctx.Done()
		closeOnce.Do(func() {
			_ = s.sock.Close()
		})
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, err := s.sock.Recv(buf)
			if err != nil {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
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

	switch pkt.Protocol {
	case ipv4.ProtocolICMP:
		return s.handleICMP(pkt, frame.SrcMAC)
	case ipv4.ProtocolUDP:
		return s.handleUDP(pkt, frame.SrcMAC)
	default:
		return nil
	}
}

func (s *Stack) handleICMP(pkt *ipv4.Packet, dstMAC net.HardwareAddr) error {
	msg, err := icmp.Parse(pkt.Payload)
	if err != nil {
		return err
	}
	if msg.IsEchoRequest() && pkt.DstIP.Equal(s.srcIP) {
		log.Printf("Ping from %s: ID=%d Seq=%d", pkt.SrcIP, msg.ID, msg.Seq)
		return s.sendEchoReply(pkt, msg, dstMAC)
	}
	return nil
}

func (s *Stack) handleUDP(pkt *ipv4.Packet, dstMAC net.HardwareAddr) error {
	msg, err := udp.Parse(pkt.Payload)
	if err != nil {
		return err
	}
	if err := udp.VerifyChecksum(pkt.Payload, pkt.SrcIP, pkt.DstIP); err != nil {
		return err
	}
	if pkt.DstIP.Equal(s.srcIP) && msg.DstPort == udp.EchoPort {
		log.Printf("UDP Echo from %s:%d (len=%d)", pkt.SrcIP, msg.SrcPort, len(msg.Data))
		return s.sendUDPReply(pkt, msg, dstMAC)
	}
	return nil
}

// shouldProcessFrame checks if we should process this frame
func (s *Stack) shouldProcessFrame(frame *ethernet.Frame) bool {
	// Process if addressed to us
	if bytes.Equal(frame.DstMAC, s.srcMAC) {
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
		Version:  4,
		IHL:      20,
		TOS:      0,
		TTL:      64,
		Protocol: ipv4.ProtocolICMP,
		SrcIP:    s.srcIP,
		DstIP:    reqPkt.SrcIP,
		Payload:  icmpData,
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

func (s *Stack) sendUDPReply(reqPkt *ipv4.Packet, reqMsg *udp.Message, dstMAC net.HardwareAddr) error {
	reply := &udp.Message{SrcPort: reqMsg.DstPort, DstPort: reqMsg.SrcPort, Data: reqMsg.Data}
	udpData, err := reply.Marshal(s.srcIP, reqPkt.SrcIP)
	if err != nil {
		return fmt.Errorf("marshal UDP: %w", err)
	}

	ipPkt := &ipv4.Packet{
		TTL: 64, Protocol: ipv4.ProtocolUDP, SrcIP: s.srcIP, DstIP: reqPkt.SrcIP, Payload: udpData,
	}
	frame := &ethernet.Frame{
		DstMAC: dstMAC, SrcMAC: s.srcMAC, EtherType: ethernet.TypeIPv4, Payload: ipPkt.Marshal(),
	}
	_, err = s.sock.Send(frame.Marshal())
	return err
}

// Close closes the stack
func (s *Stack) Close() error {
	return s.sock.Close()
}

func main() {
	iface := flag.String("iface", "", "Network interface to bind (required)")
	ip := flag.String("ip", "", "IPv4 address answered by this userspace stack (required)")
	flag.Parse()

	if *iface == "" || *ip == "" {
		fmt.Println("TCP/IP Protocol Stack Workshop")
		fmt.Println("Usage: sudo go run main.go -iface <interface> -ip <IPv4>")
		os.Exit(1)
	}

	stack, err := NewStack(*iface, net.ParseIP(*ip))
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
