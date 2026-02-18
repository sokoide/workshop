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
	"time"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ethernet"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/icmp"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/rawsock"
)

type Pinger struct {
	sock      *rawsock.Socket
	iface     string
	srcMAC    net.HardwareAddr
	srcIP     net.IP
	dstIP     net.IP
	dstMAC    net.HardwareAddr
	gatewayIP net.IP
	seq       uint16
}

func NewPinger(iface, targetIP string) (*Pinger, error) {
	sock, err := rawsock.Open(iface)
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

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
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
			srcIP = ipnet.IP
			break
		}
	}

	if srcIP == nil {
		sock.Close()
		return nil, fmt.Errorf("no IPv4 address found")
	}

	dstIP := net.ParseIP(targetIP)
	if dstIP == nil {
		sock.Close()
		return nil, fmt.Errorf("invalid target IP: %s", targetIP)
	}
	dstIP = dstIP.To4()

	// Determine destination MAC
	// Same subnet: use target MAC directly
	// Different subnet: use gateway MAC
	dstMAC, gatewayIP, err := resolveMAC(iface, srcIP, dstIP)
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("resolve MAC: %w", err)
	}

	return &Pinger{
		sock:      sock,
		iface:     iface,
		srcMAC:    ifaceObj.HardwareAddr,
		srcIP:     srcIP,
		dstIP:     dstIP,
		dstMAC:    dstMAC,
		gatewayIP: gatewayIP,
	}, nil
}

func resolveMAC(iface string, srcIP, dstIP net.IP) (net.HardwareAddr, net.IP, error) {
	// Check if same subnet
	srcMask := getLocalMask(iface)
	if srcMask != nil {
		srcNet := &net.IPNet{IP: srcIP.Mask(srcMask), Mask: srcMask}
		dstNet := &net.IPNet{IP: dstIP.Mask(srcMask), Mask: srcMask}
		if srcNet.IP.Equal(dstNet.IP) {
			// Same subnet - try to resolve target MAC via ARP
			_, err := arpResolve(iface, dstIP)
			if err == nil {
				// MAC resolved successfully
				return nil, nil, nil
			}
		}
	}

	// Different subnet or ARP failed - use gateway
	gatewayIP, _, err := getDefaultGateway(iface)
	if err != nil {
		return nil, nil, fmt.Errorf("no gateway found: %w", err)
	}

	// Resolve gateway MAC via ARP
	_, err = arpResolve(iface, gatewayIP)
	if err != nil {
		return nil, gatewayIP, fmt.Errorf("resolve gateway MAC: %w", err)
	}

	return nil, gatewayIP, nil
}

func getLocalMask(iface string) net.IPMask {
	i, err := net.InterfaceByName(iface)
	if err != nil {
		return nil
	}

	addrs, err := i.Addrs()
	if err != nil {
		return nil
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			return ipnet.Mask
		}
	}

	return nil
}

func arpResolve(iface string, targetIP net.IP) (net.HardwareAddr, error) {
	// For simplicity, return broadcast
	// In a real implementation, you would:
	// 1. Send an ARP request
	// 2. Wait for ARP reply
	// 3. Extract MAC from reply
	return nil, fmt.Errorf("ARP not implemented - use broadcast")
}

func getDefaultGateway(iface string) (net.IP, net.HardwareAddr, error) {
	// For simplicity, return default gateway MAC
	// In a real implementation, you would read /proc/net/route or use netlink
	return net.ParseIP("192.168.1.1"), net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, nil
}

func (p *Pinger) SendPing() error {
	p.seq++
	data := []byte(fmt.Sprintf("ping-%d", p.seq))

	// Build ICMP Echo Request
	icmpMsg := icmp.NewEchoRequest(12345, p.seq, data)
	icmpData := icmpMsg.Marshal()

	// Build IP packet
	ipPkt := &ipv4.Packet{
		Version:     4,
		IHL:         20,
		TOS:         0,
		TTL:         64,
		Protocol:    ipv4.ProtocolICMP,
		SrcIP:       p.srcIP,
		DstIP:       p.dstIP,
		Payload:     icmpData,
	}

	ipData := ipPkt.Marshal()

	// Build Ethernet frame
	frame := &ethernet.Frame{
		DstMAC:    p.dstMAC,
		SrcMAC:    p.srcMAC,
		EtherType: ethernet.TypeIPv4,
		Payload:   ipData,
	}

	frameData := frame.Marshal()

	_, err := p.sock.Send(frameData)
	return err
}

func (p *Pinger) Run(ctx context.Context, count int, interval time.Duration) error {
	buf := make([]byte, 65536)
	replyChan := make(chan *icmp.Message, 10)

	// Start receiver goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := p.sock.Recv(buf)
				if err != nil {
					continue
				}

				frame, err := ethernet.Parse(buf[:n])
				if err != nil {
					continue
				}

				if frame.EtherType != ethernet.TypeIPv4 {
					continue
				}

				pkt, err := ipv4.Parse(frame.Payload)
				if err != nil {
					continue
				}

				if pkt.Protocol != ipv4.ProtocolICMP {
					continue
				}

				if !pkt.SrcIP.Equal(p.dstIP) {
					continue
				}

				msg, err := icmp.Parse(pkt.Payload)
				if err != nil {
					continue
				}

				if msg.IsEchoReply() && msg.ID == 12345 {
					replyChan <- msg
				}
			}
		}
	}()

	sent := 0
	received := 0
	rtts := make([]time.Duration, 0, count)

	sendTime := make(map[uint16]time.Time)

	for i := 0; i < count; i++ {
		sendTime[p.seq+1] = time.Now()

		if err := p.SendPing(); err != nil {
			log.Printf("Send failed: %v", err)
		} else {
			sent++
			fmt.Printf("Sent ping to %s: seq=%d\n", p.dstIP, p.seq)
		}

		select {
		case msg := <-replyChan:
			received++
			rtt := time.Since(sendTime[msg.Seq])
			rtts = append(rtts, rtt)
			fmt.Printf("Reply from %s: seq=%d time=%v\n", p.dstIP, msg.Seq, rtt)
		case <-time.After(interval):
			fmt.Printf("Timeout: seq=%d\n", p.seq)
		case <-ctx.Done():
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Wait for remaining replies
	time.Sleep(1 * time.Second)

	// Print statistics
	fmt.Println("\n--- Statistics ---")
	fmt.Printf("Packets: Sent=%d, Received=%d, Lost=%d (%.1f%% loss)\n",
		sent, received, sent-received,
		float64(sent-received)/float64(sent)*100)

	if len(rtts) > 0 {
		var min, max, sum time.Duration
		min = rtts[0]
		max = rtts[0]
		for _, rtt := range rtts {
			if rtt < min {
				min = rtt
			}
			if rtt > max {
				max = rtt
			}
			sum += rtt
		}
		avg := sum / time.Duration(len(rtts))
		fmt.Printf("RTT: min=%v, max=%v, avg=%v\n", min, max, avg)
	}

	return nil
}

func (p *Pinger) Close() error {
	return p.sock.Close()
}

func main() {
	iface := flag.String("iface", "", "Network interface (required)")
	target := flag.String("target", "", "Target IP address (required)")
	count := flag.Int("count", 4, "Number of pings to send")
	interval := flag.Duration("interval", time.Second, "Interval between pings")
	flag.Parse()

	if *iface == "" || *target == "" {
		fmt.Println("Custom Ping Implementation")
		fmt.Println("Usage: sudo go run main.go -iface <interface> -target <ip> [options]")
		fmt.Println("")
		fmt.Println("Example:")
		fmt.Println("  sudo go run main.go -iface eth0 -target 192.168.1.1")
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	pinger, err := NewPinger(*iface, *target)
	if err != nil {
		log.Fatalf("NewPinger: %v", err)
	}
	defer pinger.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("Pinging %s via %s", *target, *iface)

	if err := pinger.Run(ctx, *count, *interval); err != nil {
		log.Fatalf("Run: %v", err)
	}
}
