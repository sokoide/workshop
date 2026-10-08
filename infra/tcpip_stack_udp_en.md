# TCP/IP Protocol Stack Workshop — Part 2: UDP & Stack Integration

> **⏱️ Estimated Time**: Approx. 80 minutes (Part 2 of 2; the complete workshop takes approx. 180 minutes)
> **🎯 Level**: Advanced
> **💻 Environment**: Linux (root/sudo)

This is Part 2 of the two-part TCP/IP Protocol Stack Workshop. [Part 1: Ethernet, IP & ICMP](tcpip_stack_en.md) built the AF_PACKET raw socket and the Ethernet, IPv4, and ICMP layers. In this part you implement UDP datagrams, integrate all components into `main.go`, and run the live ping and UDP echo tests against your own stack.

> **💡 Glossary**: Please refer to [Raw Socket](glossary_en.md#protocol) or [Encapsulation](glossary_en.md#protocol) in the [Glossary](glossary_en.md) for technical terms used in this workshop.

## Goal

Implement the UDP layer (`pkg/udp`), wire every component into `main.go`, and verify your own stack with live traffic: a ping from inside the namespace gets replies from `192.168.100.1`, and the UDP echo test returns `b'hello udp'` from your stack.

## Prerequisites

- [ ] Completed [Part 1: Ethernet, IP & ICMP](tcpip_stack_en.md) — the `pkg/rawsock`, `pkg/ethernet`, `pkg/ipv4`, and `pkg/icmp` packages exist in the project created there
- [ ] The Part 1 sandbox is still up: `ip netns list` shows `workshop`, `veth-host` has no IPv4 address, and the namespace holds the permanent neighbor entry for `192.168.100.1` (otherwise redo Part 1's Environment Setup)
- [ ] Go 1.22 or later and GCC with CGO enabled (installed in Part 1)

---

## 🚀 Implementation Steps

The step numbers continue from Part 1, which ended at STEP 4.

### STEP 5: UDP Message (L4) — Connectionless Communication

#### Goal

Implement UDP to carry application data without connection setup. Contrast it with the TCP handshake observed in STEP 0 and the diagnostic role of ICMP Echo.

#### How UDP Differs from TCP and ICMP

| Property                         | ICMP                | UDP                               | TCP                              |
| :------------------------------- | :------------------ | :-------------------------------- | :------------------------------- |
| Connection setup                 | None                | None                              | Handshake                        |
| Ports                            | None                | 16-bit ports                      | 16-bit ports                     |
| Delivery and ordering guarantees | None                | None                              | Retransmission and ordering      |
| Checksum coverage                | Entire ICMP message | Pseudo-header and entire datagram | Pseudo-header and entire segment |
| Examples                         | Ping                | DNS, DHCP, metrics                | HTTP/1.1, SSH                    |

The pseudo-header helps detect accidental misdelivery. It does not authenticate a sender who can recompute the checksum.

#### UDP Datagram Structure

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Source Port           |      Destination Port         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Length             |          Checksum             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                             Data                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

The header is eight bytes. Length includes both header and data.

#### Implementation: UDP Message (`pkg/udp/message.go`)

Read the [executable UDP implementation](assets/tcpip_stack/pkg/udp/message.go) and its [tests](assets/tcpip_stack/pkg/udp/message_test.go). `Parse`, `VerifyChecksum`, and `Marshal` separate length validation, checksum verification, and reply construction. Use this source directly to avoid maintaining a second copy of the implementation in the tutorial.

#### Key Learning Points

##### The Pseudo-Header's Role

Checksum input includes source and destination IPv4 addresses, a zero byte, protocol number 17, and UDP length, followed by the datagram. A computed checksum of zero is transmitted as `0xffff`; a transmitted zero means the IPv4 sender omitted the checksum.

##### What Connectionless Means

UDP sends with `sendto()` without a transport handshake. The receiver obtains the reply address from the packet's source IP and UDP source port.

##### Why Use UDP?

DNS can avoid transport connection setup for a small query. DHCP needs to communicate before normal address configuration. Metrics and real-time media may prefer timely delivery over retransmission. Applications must implement any reliability they require.

##### Contrast with TCP

TCP additionally requires connection state, sequence numbers, acknowledgements, retransmission, flow control, and congestion control. UDP leaves these policies to the application or a higher-level protocol.

#### ✅ Verification Checkpoints

- [ ] `HeaderSize` is 8.
- [ ] `Marshal` uses the source and destination IPs in its checksum.
- [ ] A computed zero checksum is encoded as `0xffff`.
- [ ] Invalid lengths and corrupted checksums are rejected before replying.
- [ ] The stack dispatches ICMP and UDP separately.

---

### STEP 6: Stack Integration (`main.go`) - Connecting Everything

#### Goal

Integrate all components — including the UDP datagrams from the previous step — to build a server that responds to Ping and UDP Echo requests.

#### Implementation: Main Program (`main.go`)

The listing below matches [assets/tcpip_stack/main.go](assets/tcpip_stack/main.go) except for the `//go:build linux` build tag on its first line (this workshop is Linux-only anyway, but keep the tag when you copy the file).

```go
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
	// Reassembly is outside this workshop's scope. Do not treat a fragment as
	// a complete ICMP message or UDP datagram.
	if pkt.IsFragmented() {
		return fmt.Errorf("fragmented IPv4 packets are unsupported")
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
```

---

## 🧪 Testing

### 1. Build

```bash
go mod tidy
go build -o tcpip_stack main.go
```

### 2. Start the Stack

```bash
# Start on the host side of the created veth
sudo ./tcpip_stack -iface veth-host -ip 192.168.100.1
```

### 3. Ping Test

From another terminal:

```bash
# Ping from within the namespace
sudo ip netns exec workshop ping 192.168.100.1
# Expected output: 64 bytes from 192.168.100.1: icmp_seq=1 ttl=64 (one line per reply)
```

### 4. Observe Packets

```bash
sudo tcpdump -i veth-host -nn -vv icmp
# Expected output: IP 192.168.100.2 > 192.168.100.1: ICMP echo request ... / IP 192.168.100.1 > 192.168.100.2: ICMP echo reply ...
```

### 5. UDP Echo Test

```bash
sudo ip netns exec workshop python3 - <<'PY'
import socket
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.settimeout(1)
sock.sendto(b"hello udp", ("192.168.100.1", 7))
data, peer = sock.recvfrom(1024)
assert data == b"hello udp", (data, peer)
print(f"reply={data!r} from {peer}")
# Expected output: reply=b'hello udp' from ('192.168.100.1', 7)
PY

sudo tcpdump -i veth-host -nn -vv -XX 'udp port 7'
```

---

### 6. Observe and Corrupt UDP

Use `tcpdump -XX` to map the eight-byte UDP header and payload to the captured bytes. Then run the `pkg/udp` tests: flipping a checksum bit must make `VerifyChecksum` reject the datagram. Verify both accepted and rejected messages.

---

## What you learned

- How the kernel encapsulates application data into Ethernet (L2), IPv4 (L3), and ICMP/UDP (L4), and how to parse each header from raw bytes.
- How to use Linux `AF_PACKET` raw sockets (via CGO) to observe and inject frames on a veth interface.
- How network namespaces and veth pairs provide an isolated sandbox for low-level networking experiments.
- How the RFC 1071 checksum (including the UDP pseudo-header) and network byte order (Big Endian) protect packet integrity.
- Why a full TCP implementation (handshake, sequencing, retransmission, congestion control) is a project of its own — see Deep Dive Topics below.

**Next step**: See how real applications reuse and upgrade these connections in the [HTTP Persistent Connection Workshop](http_persistent_en.md).

---

## 📊 Packet Flow Diagram

```text
=================== Receive Path (Rx) ===================

NIC --> Raw Socket --> Ethernet.Parse --> IPv4.Parse --> ICMP.Parse
                           |  |  |
                      [MAC Filter]      [IP Filter]    [Type Check]
                           |  |  |
                        Pass?              Pass?           Type=8?
                           |  |  |
                           v                  v               v
                      Drop/Continue     Drop/Continue   Drop/Reply

==================== Send Path (Tx) ====================

NIC <-- Raw Socket <-- Ethernet.Marshal <-- IPv4.Marshal <-- ICMP.Reply
```

**Legend:**

- `-->` : Data flow (packet/byte sequence)
- `|` : Processing branch point
- `v` : Next processing step
- `[]` : Filtering process

### Detailed Packet Processing Flow

#### 1. Receive Path (Left to Right)

Processing flow when a Ping request arrives:

```text
Step 1: NIC → Raw Socket
────────────────────────
・NIC converts electrical signals to bit sequence
・Raw Socket receives as byte sequence
・Data: Raw bytes of [Ethernet][IPv4][ICMP][Data]

Step 2: Ethernet.Parse
────────────────────────
・Parse Ethernet header from byte sequence
・Extract: DstMAC, SrcMAC, EtherType
・[MAC Filter]: Check if DstMAC is addressed to us
  - Our MAC or Broadcast → Continue
  - Otherwise → Drop (end processing)

Step 3: IPv4.Parse
────────────────────────
・Parse IPv4 header from Ethernet payload
・Extract: SrcIP, DstIP, Protocol, TTL, etc.
・[IP Filter]: Check if Protocol is ICMP
  - Protocol = 1 (ICMP) → Continue
  - Otherwise → Drop (TCP/UDP ignored in this implementation)

Step 4: ICMP.Parse
────────────────────────
・Parse ICMP message from IPv4 payload
・Extract: Type, Code, ID, Seq, Data
・[Type Check]: Check if Type is Echo Request (8)
  - Type = 8 → Generate Echo Reply and send
  - Otherwise → Drop
```

#### 2. Send Path (Right to Left)

Processing flow when sending Echo Reply:

```text
Step 1: ICMP.Reply Generation
─────────────────────────────
・Create Reply from received Echo Request
・Type: Change 8 → 0
・ID, Seq, Data: Copy as-is
・Checksum: Recalculate

Step 2: IPv4.Marshal
────────────────────────
・Convert IPv4 packet to byte sequence
・Swap SrcIP/DstIP (requester → us)
・TTL: Set to 64
・Checksum: Calculate and set

Step 3: Ethernet.Marshal
────────────────────────
・Convert Ethernet frame to byte sequence
・Swap DstMAC/SrcMAC (requester → us)
・EtherType: 0x0800 (IPv4)

Step 4: Raw Socket → NIC
────────────────────────
・Pass completed byte sequence to Raw Socket
・NIC transmits as electrical signal
```

### Concrete Example: Ping Request-Reply Sequence

```text
T1: Execute ping 192.168.100.1 from Namespace
    |
    v
T2: Packet arrives (Echo Request)
    +---------------------------------------------------+
    | Ethernet: Dst=<veth-host MAC>                  |
    | IPv4:     Src=192.168.100.2, Dst=192.168.100.1 |
    | ICMP:     Type=8, ID=1234, Seq=1               |
    +---------------------------------------------------+
    |
    v  Arrives at veth-host
    |
T3: Custom stack receives
    | -> MAC Filter: our MAC -> Pass
    | -> IP Filter:  DstIP is ours -> Pass
    | -> Type Check: Type=8 -> Generate Reply
    |
    v
T4: Packet sent (Echo Reply)
    +---------------------------------------------------+
    | Ethernet: Dst=xx:xx:xx:xx:xx:xx                |
    | IPv4:     Src=192.168.100.1, Dst=192.168.100.2 |
    | ICMP:     Type=0, ID=1234, Seq=1               |
    +---------------------------------------------------+
    |
    v  Send to Namespace
    |
T5: ping receives Reply
    "64 bytes from 192.168.100.1: icmp_seq=1 ttl=64"
```

---

## 🧹 Cleanup

These commands remove the sandbox created in Part 1:

```bash
sudo ip link delete veth-host
sudo ip netns delete workshop
```

---

## 💡 Deep Dive Topics

1. **Extensibility**: Design considerations for adding TCP on top of this stack — connection state, sequencing, retransmission, and congestion control are exactly what this workshop defers by stopping at ICMP and UDP

## 📚 References

- [RFC 768 (UDP)](https://datatracker.ietf.org/doc/html/rfc768)
- [RFC 862 (Echo Protocol)](https://datatracker.ietf.org/doc/html/rfc862)
- [RFC 793 (TCP)](https://datatracker.ietf.org/doc/html/rfc793) — not implemented in this workshop, but useful reading next to UDP
- [lwIP - A Lightweight TCP/IP Stack](https://savannah.nongnu.org/projects/lwip/) — educational reference for a full TCP implementation

---

## 🔧 Troubleshooting

For raw socket and build errors, see the Troubleshooting section of [Part 1](tcpip_stack_en.md).

### UDP Echo Times Out / "invalid UDP checksum"

If `tcpdump` shows `[bad udp cksum ...]` on UDP frames, the veth TX checksum offload is sending CHECKSUM_PARTIAL frames — their checksum field is not final, so the userspace stack (raw socket) rightly rejects them. `make setup-env` disables the offload with `ethtool -K ... tx off`; if you created the veth manually, run:

```bash
sudo ethtool -K veth-host tx off rx off tso off gso off gro off
sudo ip netns exec workshop ethtool -K veth-ns tx off rx off tso off gso off gro off
```

### Ping Does Not Work

Ensure the stack is running, inspect the interface, and confirm the namespace's permanent neighbor entry from setup.

```bash
ip link show veth-host
ip addr show veth-host
sudo ip netns exec workshop ip neigh show
sudo ip netns exec workshop ping 192.168.100.1
sudo tcpdump -i veth-host -nn -vv icmp
```
