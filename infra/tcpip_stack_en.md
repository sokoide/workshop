# TCP/IP Protocol Stack Workshop: Building the "Soul of Communication" from Scratch

In this workshop, you will combine Go and C to **build a TCP/IP protocol stack from the ground up**. By implementing the "packet generation and decomposition" that OS kernels usually hide, you will understand network principles not as abstract concepts, but as concrete data structures.

> **💡 Glossary**: Please refer to [Raw Socket](glossary_en.md#protocol) or [Encapsulation](glossary_en.md#protocol) in the [Glossary](glossary_en.md) for technical terms used in this workshop.

---

## 🏗 Architecture: Russian Doll (Encapsulation)

The essence of network communication is "Encapsulation." Higher-layer data is wrapped as the "payload" of lower layers.

```text
+----------------------------------------------------+
| Ethernet Frame (L2)                                |
| +------------------------------------------------+ |
|                                                    | IPv4 Packet (L3)                               |                                            |
|                                                    | +--------------------------------------------+ |                                            |
|                                                    |                                                | ICMP Message (L4)                          |                |  |
|                                                    |                                                | +----------------------------------------+ |                |  |
|                                                    |                                                |                                            | Payload (Data) |  |  |  |
|                                                    |                                                | +----------------------------------------+ |                |  |
|                                                    | +--------------------------------------------+ |                                            |
| +------------------------------------------------+ |
+----------------------------------------------------+
```

### Components to Implement

| Layer | Component      | File                    | Description                               |
| :---- | :------------- | :---------------------- | :---------------------------------------- |
| L2    | Packet Socket  | `pkg/rawsock/`          | Linux AF_PACKET access to Ethernet frames |
| L2    | Ethernet Frame | `pkg/ethernet/frame.go` | MAC address-based frame handling          |
| L3    | IPv4 Packet    | `pkg/ipv4/packet.go`    | IP address-based packet handling          |
| L4    | ICMP Message   | `pkg/icmp/message.go`   | Ping (Echo Request/Reply) implementation  |
| L4    | UDP Message    | `pkg/udp/message.go`    | Connectionless application datagrams      |
| App   | Stack          | `main.go`               | Server integrating all components         |

---

## 🛠 Environment Setup

This workshop requires **Linux (Ubuntu 22.04/24.04 recommended)** as it involves low-level Raw Socket operations.

### 1. Required Tools

```bash
sudo apt update && sudo apt install -y golang-go gcc tcpdump wireshark iproute2
```

### 2. Project Structure

```bash
mkdir -p tcpip_stack/pkg/{rawsock,ethernet,ipv4,icmp,udp}
mkdir -p tcpip_stack/cmd/ping
cd tcpip_stack
go mod init github.com/sokoide/workshop/infra/assets/tcpip_stack
```

### 3. Creating a "Safe" Sandbox (Network Namespace)

When developing with Raw Sockets, sending incorrect packets risks confusing the host OS routing table. We use Linux **Network Namespaces (netns)** to build a logically isolated "dedicated sandbox."

```bash
# 1. Create an isolated "sandbox" (Namespace: workshop)
sudo ip netns add workshop

# 2. Create a virtual LAN cable (veth) with endpoints veth-host and veth-ns
sudo ip link add veth-host type veth peer name veth-ns

# 3. Move one end (veth-ns) to the isolation space
sudo ip link set veth-ns netns workshop

# 4. Bring both interfaces up (link up)
sudo ip link set veth-host up
sudo ip netns exec workshop ip link set veth-ns up

# 5. Assign an IP only to the client. The userspace stack owns .1, so Linux
# must not also own and answer for it.
sudo ip netns exec workshop ip addr add 192.168.100.2/24 dev veth-ns

# 6. Skip ARP implementation for now with a static neighbor entry.
STACK_MAC=$(cat /sys/class/net/veth-host/address)
sudo ip netns exec workshop ip neigh replace 192.168.100.1 lladdr "$STACK_MAC" nud permanent dev veth-ns
```

### ✅ Verification Checkpoints

- [ ] `workshop` netns exists via `ip netns list`.
- [ ] `veth-host` has no IPv4 address.
- [ ] `192.168.100.2` is assigned to `veth-ns` in the namespace.
- [ ] A permanent neighbor entry exists for `192.168.100.1`.

---

## 🚀 Implementation Steps

### STEP 0: Observe Communication — What the Kernel Hides

#### Goal

Before implementing packets, compare an application's HTTP exchange with the packets sent by the kernel.

#### 1. Communication from the Application's View (curl -v)

Start a local server in another terminal, then make a request.

```bash
python3 -m http.server 8000
# In the client terminal:
curl -v http://localhost:8000/
```

The output shows connection status, request and response headers, and the response body. It does not show individual TCP segments or their headers.

#### 2. What the Kernel Actually Sends (tcpdump)

Capture the same request on Linux loopback.

```bash
sudo tcpdump -i lo -nn -vv 'tcp port 8000'
# In the client terminal:
curl -s http://localhost:8000/ > /dev/null
```

Look for `[S]`, `[S.]`, and `[.]`: SYN, SYN-ACK, and ACK establish the TCP connection. Data, acknowledgements, and connection teardown follow. The loopback capture shows IP/TCP traffic; Ethernet framing is examined later on the veth interfaces.

#### Key Learning Points

- The socket API hides packet exchange from the application.
- TCP maintains sequence numbers, acknowledgements, retransmission, and connection state. Implementing all of TCP is beyond this workshop.
- Building ICMP Echo and UDP demonstrates packet structure and encapsulation with fewer protocol states.

#### ✅ Verification Checkpoints

- [ ] Find the `Connected` line in `curl -v` output.
- [ ] Identify the three-way handshake in `tcpdump`.
- [ ] Confirm that one HTTP request involves multiple packets.

Stop the HTTP server with Ctrl-C in its terminal.

---

### STEP 1: Packet Socket (L2) - Observe and Send Ethernet Frames

#### Goal

Normal applications use `TCP/UDP Sockets`, so the OS generates L2/L3 headers. This workshop uses **AF_PACKET** to receive Ethernet frames alongside the normal Linux IP stack and to build headers itself. It does not bypass the NIC driver or the whole kernel.

#### Implementation: C Header File (`pkg/rawsock/rawsock.h`)

```c
#ifndef RAWSOCK_H
#define RAWSOCK_H

#ifdef __cplusplus
extern "C" {
#endif

// Open raw socket on specified interface
// Returns file descriptor on success, -1 on failure
int rawsock_open(const char *iface);

// Receive packet into buffer
// Returns number of bytes received, -1 on error
int rawsock_recv(int fd, void *buf, int len);

// Send packet on specified interface
// Returns number of bytes sent, -1 on error
int rawsock_send(int fd, const void *buf, int len, const char *iface);

// Close raw socket
void rawsock_close(int fd);

// Get interface MAC address
// Returns 0 on success, -1 on failure
int rawsock_get_mac(const char *iface, unsigned char *mac);

// Get interface MTU
// Returns MTU on success, -1 on failure
int rawsock_get_mtu(const char *iface);

// Set promiscuous mode
// Returns 0 on success, -1 on failure
int rawsock_set_promisc(int fd, const char *iface, int enable);

#ifdef __cplusplus
}
#endif

#endif // RAWSOCK_H
```

#### Implementation: C Source File (`pkg/rawsock/rawsock.c`)

```c
#include "rawsock.h"
#include <sys/socket.h>
#include <sys/ioctl.h>
#include <linux/if_ether.h>
#include <linux/if_packet.h>
#include <net/if.h>
#include <unistd.h>
#include <string.h>
#include <arpa/inet.h>
#include <stdio.h>

int rawsock_open(const char *iface) {
    // AF_PACKET: Direct access to Data Link layer (L2)
    // SOCK_RAW: Receive all packets including headers
    // ETH_P_ALL: Receive all protocol types
    int fd = socket(AF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
    if (fd < 0) {
        perror("socket");
        return -1;
    }

    // Bind to specified interface
    struct sockaddr_ll addr;
    memset(&addr, 0, sizeof(addr));
    addr.sll_family = AF_PACKET;
    addr.sll_protocol = htons(ETH_P_ALL);
    addr.sll_ifindex = if_nametoindex(iface);

    if (addr.sll_ifindex == 0) {
        perror("if_nametoindex");
        close(fd);
        return -1;
    }

    if (bind(fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("bind");
        close(fd);
        return -1;
    }

    return fd;
}

int rawsock_recv(int fd, void *buf, int len) {
    return recv(fd, buf, len, 0);
}

int rawsock_send(int fd, const void *buf, int len, const char *iface) {
    struct sockaddr_ll addr;
    memset(&addr, 0, sizeof(addr));
    addr.sll_family = AF_PACKET;
    addr.sll_ifindex = if_nametoindex(iface);

    if (addr.sll_ifindex == 0) {
        return -1;
    }

    return sendto(fd, buf, len, 0, (struct sockaddr*)&addr, sizeof(addr));
}

void rawsock_close(int fd) {
    if (fd >= 0) {
        close(fd);
    }
}

int rawsock_get_mac(const char *iface, unsigned char *mac) {
    int fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) {
        return -1;
    }

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, iface, IFNAMSIZ - 1);

    // Get hardware address via ioctl
    if (ioctl(fd, SIOCGIFHWADDR, &ifr) < 0) {
        perror("ioctl SIOCGIFHWADDR");
        close(fd);
        return -1;
    }

    memcpy(mac, ifr.ifr_hwaddr.sa_data, 6);
    close(fd);
    return 0;
}

int rawsock_get_mtu(const char *iface) {
    int fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) {
        return -1;
    }

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, iface, IFNAMSIZ - 1);

    if (ioctl(fd, SIOCGIFMTU, &ifr) < 0) {
        close(fd);
        return -1;
    }

    int mtu = ifr.ifr_mtu;
    close(fd);
    return mtu;
}

int rawsock_set_promisc(int fd, const char *iface, int enable) {
    struct packet_mreq mr;
    memset(&mr, 0, sizeof(mr));
    mr.mr_ifindex = if_nametoindex(iface);
    mr.mr_type = PACKET_MR_PROMISC;

    if (setsockopt(fd, SOL_PACKET, PACKET_ADD_MEMBERSHIP, &mr, sizeof(mr)) < 0) {
        perror("setsockopt PACKET_ADD_MEMBERSHIP");
        return -1;
    }

    return 0;
}
```

#### Implementation: Go Bindings (`pkg/rawsock/rawsock.go`)

```go
package rawsock

/*
#cgo LDFLAGS: -lpcap
#include "rawsock.h"
*/
import "C"
import (
	"fmt"
	"net"
	"unsafe"
)

// Socket represents a raw socket
type Socket struct {
	fd    C.int
	iface string
}

// Open creates a new raw socket on the specified interface
func Open(iface string) (*Socket, error) {
	cIface := C.CString(iface)
	defer C.free(unsafe.Pointer(cIface))

	fd := C.rawsock_open(cIface)
	if fd < 0 {
		return nil, fmt.Errorf("failed to open raw socket on %s", iface)
	}

	return &Socket{
		fd:    fd,
		iface: iface,
	}, nil
}

// Recv receives a packet into the provided buffer
// Important: Uses unsafe.Pointer to write directly to Go slice, avoiding copies
func (s *Socket) Recv(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	n := C.rawsock_recv(s.fd, unsafe.Pointer(&buf[0]), C.int(len(buf)))
	if n < 0 {
		return 0, fmt.Errorf("recv failed")
	}
	return int(n), nil
}

// Send sends a packet on the interface
func (s *Socket) Send(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	cIface := C.CString(s.iface)
	defer C.free(unsafe.Pointer(cIface))

	n := C.rawsock_send(s.fd, unsafe.Pointer(&buf[0]), C.int(len(buf)), cIface)
	if n < 0 {
		return 0, fmt.Errorf("send failed")
	}
	return int(n), nil
}

// Close closes the raw socket
func (s *Socket) Close() error {
	C.rawsock_close(s.fd)
	s.fd = -1
	return nil
}

// GetMAC returns the MAC address of the interface
func GetMAC(iface string) (net.HardwareAddr, error) {
	cIface := C.CString(iface)
	defer C.free(unsafe.Pointer(cIface))

	var mac [6]C.uchar
	ret := C.rawsock_get_mac(cIface, &mac[0])
	if ret < 0 {
		return nil, fmt.Errorf("failed to get MAC address")
	}

	return net.HardwareAddr{
		byte(mac[0]), byte(mac[1]), byte(mac[2]),
		byte(mac[3]), byte(mac[4]), byte(mac[5]),
	}, nil
}
```

#### Key Learning Points

##### What is AF_PACKET?

AF_PACKET is a Linux socket family for **direct access to the Data Link Layer (L2)**. With regular sockets (AF_INET), the kernel automatically handles TCP/IP headers. Using AF_PACKET, you can send/receive **the entire Ethernet frame as raw bytes**.

```text
Regular Socket (AF_INET)
+-----------+     +------------------------+
| App | <-- | Kernel handles TCP/IP |
+-----------+     +------------------------+

Raw Socket (AF_PACKET)
+-----------+     +------------------------------------+
| App | <-- | Raw Ethernet frame (bytes) |
+-----------+     | Build headers from MAC layer up    |
                  +------------------------------------+
```

##### socket() System Call Arguments

```c
int fd = socket(AF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
//              │          │         │
//              │          │         └─ Protocol: Receive all packets
//              │          └─ Type: Raw data including headers
//              └─ Family: Data link layer access
```

| Argument | Value                | Meaning                           |
| :------- | :------------------- | :-------------------------------- |
| Domain   | `AF_PACKET`          | Packet-level socket (L2)          |
| Type     | `SOCK_RAW`           | Get raw packets including headers |
| Protocol | `ETH_P_ALL` (0x0003) | Receive all protocol types        |

##### Other Protocol Values

| Protocol     | Value  | Description         |
| :----------- | :----- | :------------------ |
| `ETH_P_ALL`  | 0x0003 | Receive all packets |
| `ETH_P_IP`   | 0x0800 | Receive IPv4 only   |
| `ETH_P_IPV6` | 0x86DD | Receive IPv6 only   |
| `ETH_P_ARP`  | 0x0806 | Receive ARP only    |

##### Why Do We Need AF_PACKET?

When building a custom stack, AF_PACKET is essential because:

1. **Custom Header Generation**: Build Ethernet, IP, ICMP headers yourself
2. **Bypass Kernel**: Communicate directly with NIC without kernel network stack
3. **Custom Protocols**: Implement non-standard protocols
4. **Packet Analysis**: Build network monitors like Wireshark

##### Important Notes

- **Root Privileges Required**: `CAP_NET_RAW` capability needed for security reasons
- **Linux Only**: macOS uses BPF (Berkeley Packet Filter), Windows uses WinPcap/Npcap
- **Risk of Mis-sending**: Sending wrong packets may affect the entire network

##### Role of CGO

CGO is used to call C system calls directly from Go:

```go
/*
#cgo LDFLAGS: -lpcap
#include "rawsock.h"
*/
import "C"

// Pass Go memory directly to C using unsafe.Pointer (avoid copy)
n := C.rawsock_recv(s.fd, unsafe.Pointer(&buf[0]), C.int(len(buf)))
```

- **Zero-Copy**: Pass Go slice address directly to C using `unsafe.Pointer`, avoiding memory copy
- **Performance**: Microsecond-level speed is critical in packet processing

##### Promiscuous Mode

Normally, NIC only passes packets addressed to its own MAC or broadcast. Enabling promiscuous mode allows receiving **all packets**.

```c
setsockopt(fd, SOL_PACKET, PACKET_ADD_MEMBERSHIP, &mr, sizeof(mr));
```

---

### ✅ Verification Checkpoints

- [ ] `pkg/rawsock/rawsock.c` calls `socket(AF_PACKET, SOCK_RAW, ...)`.
- [ ] Opening a raw socket fails without root or the required capability.

---

### STEP 2: Ethernet Frame (L2) - MAC Address-Based Communication

#### Goal

Ethernet controls communication with physically adjacent devices using MAC addresses (48-bit) to identify communication partners within the same segment.

#### Ethernet II Frame Structure

```text
┌───────────────────┬───────────────────┬──────────────┬─────────────┐
│  Dst MAC (6bytes) │  Src MAC (6bytes) │ Type (2B)    │ Payload     │
├───────────────────┴───────────────────┴──────────────┴─────────────┤
│                          Minimum 60 bytes                          │
└────────────────────────────────────────────────────────────────────┘
```

#### Implementation: Ethernet Frame (`pkg/ethernet/frame.go`)

```go
package ethernet

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// Ethernet II header size
	HeaderSize = 14 // 6 (DstMAC) + 6 (SrcMAC) + 2 (EtherType)

	// Minimum frame size (including padding)
	MinFrameSize = 60

	// Maximum frame size
	MaxFrameSize = 1514

	// EtherType values
	TypeIPv4 = 0x0800
	TypeARP  = 0x0806
	TypeIPv6 = 0x86DD
)

// Frame represents an Ethernet II frame
type Frame struct {
	DstMAC    net.HardwareAddr
	SrcMAC    net.HardwareAddr
	EtherType uint16
	Payload   []byte
}

// Parse parses an Ethernet II frame from bytes
func Parse(data []byte) (*Frame, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("packet too short: %d < %d", len(data), HeaderSize)
	}

	return &Frame{
		DstMAC:    net.HardwareAddr(data[0:6]),                    // Destination MAC
		SrcMAC:    net.HardwareAddr(data[6:12]),                   // Source MAC
		EtherType: binary.BigEndian.Uint16(data[12:14]),           // Protocol type
		Payload:   data[14:],                                      // Payload (upper layer data)
	}, nil
}

// Marshal converts the frame to bytes
func (f *Frame) Marshal() []byte {
	payloadLen := len(f.Payload)
	totalLen := HeaderSize + payloadLen

	buf := make([]byte, totalLen)

	// Copy MAC addresses
	copy(buf[0:6], f.DstMAC)
	copy(buf[6:12], f.SrcMAC)

	// Set EtherType (Network Byte Order = Big Endian)
	binary.BigEndian.PutUint16(buf[12:14], f.EtherType)

	// Copy payload
	if payloadLen > 0 {
		copy(buf[14:], f.Payload)
	}

	// Pad to minimum frame size if needed
	// Ethernet physical layer cannot correctly handle frames < 60 bytes
	if totalLen < MinFrameSize {
		padded := make([]byte, MinFrameSize)
		copy(padded, buf)
		return padded
	}

	return buf
}

// BroadcastMAC returns the broadcast MAC address
func BroadcastMAC() net.HardwareAddr {
	return net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
}

// IsBroadcast checks if the MAC address is broadcast
func IsBroadcast(mac net.HardwareAddr) bool {
	if len(mac) != 6 {
		return false
	}
	for _, b := range mac {
		if b != 0xFF {
			return false
		}
	}
	return true
}

// IsMulticast checks if the MAC address is multicast
// Multicast MAC has the least significant bit set to 1
func IsMulticast(mac net.HardwareAddr) bool {
	if len(mac) != 6 {
		return false
	}
	return mac[0]&0x01 == 0x01
}
```

#### Key Learning Points

- **Network Byte Order**: Big Endian is used in network communication
- **Padding**: Ethernet frames must be at least 60 bytes. Pad with zeros if shorter
- **EtherType**: Identifies the payload protocol (0x0800 = IPv4)

---

### STEP 3: IPv4 Packet (L3) - IP Address-Based Routing

#### Goal

IPv4 enables communication between different network segments. It uses 32-bit IP addresses to identify destinations and checksums to verify header integrity.

#### IPv4 Header Structure

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Version | IHL | TOS | Total Length |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Identification | Flags | Fragment Offset |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| TTL | Protocol | Header Checksum |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Source Address |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Destination Address |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

#### Implementation: IPv4 Packet (`pkg/ipv4/packet.go`)

```go
package ipv4

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// HeaderSize is the minimum IPv4 header size
	HeaderSize = 20

	// MaxPacketSize is the maximum IPv4 packet size (65535)
	MaxPacketSize = 65535

	// Protocol numbers
	ProtocolICMP = 1
	ProtocolTCP  = 6
	ProtocolUDP  = 17
	ProtocolIPv6 = 41

	// Flags
	FlagDF = 0x02 // Don't Fragment (wire bit 14)
	FlagMF = 0x01 // More Fragments (wire bit 13)
)

// Packet represents an IPv4 packet
type Packet struct {
	Version        uint8
	IHL            uint8 // Internet Header Length in 32-bit words
	TOS            uint8 // Type of Service
	TotalLength    uint16
	ID             uint16
	Flags          uint8
	FragmentOffset uint16 // In 8-byte units
	TTL            uint8  // Time to Live
	Protocol       uint8
	Checksum       uint16
	SrcIP          net.IP
	DstIP          net.IP
	Options        []byte
	Payload        []byte
}

// Parse parses an IPv4 packet from bytes
func Parse(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("packet too short: %d < %d", len(data), HeaderSize)
	}

	version := data[0] >> 4
	if version != 4 {
		return nil, fmt.Errorf("not IPv4: version=%d", version)
	}

	ihl := (data[0] & 0x0F) * 4
	if ihl < HeaderSize {
		return nil, fmt.Errorf("invalid IHL: %d < %d", ihl, HeaderSize)
	}
	if len(data) < int(ihl) {
		return nil, fmt.Errorf("packet too short for IHL: %d < %d", len(data), ihl)
	}
	totalLen := binary.BigEndian.Uint16(data[2:4])
	if totalLen < uint16(ihl) {
		return nil, fmt.Errorf("invalid total length: %d < ihl=%d", totalLen, ihl)
	}
	if int(totalLen) > len(data) {
		return nil, fmt.Errorf("packet too short for total length: %d < %d", len(data), totalLen)
	}
	if Checksum(data[:ihl]) != 0 {
		return nil, fmt.Errorf("invalid IPv4 header checksum")
	}

	flagsFragment := binary.BigEndian.Uint16(data[6:8])
	pkt := &Packet{
		Version:        version,
		IHL:            ihl,
		TOS:            data[1],
		TotalLength:    totalLen,
		ID:             binary.BigEndian.Uint16(data[4:6]),
		Flags:          uint8((flagsFragment >> 13) & 0x07),
		FragmentOffset: flagsFragment & 0x1FFF,
		TTL:            data[8],
		Protocol:       data[9],
		Checksum:       binary.BigEndian.Uint16(data[10:12]),
		SrcIP:          net.IP(data[12:16]),
		DstIP:          net.IP(data[16:20]),
		Payload:        data[ihl:totalLen],
	}

	if ihl > HeaderSize {
		pkt.Options = data[HeaderSize:ihl]
	}

	return pkt, nil
}

// Marshal converts the packet to bytes
func (p *Packet) Marshal() []byte {
	ihl := HeaderSize
	if len(p.Options) > 0 {
		// IHL is in 32-bit words
		ihl = HeaderSize + (len(p.Options)+3)/4*4
	}

	payloadLen := len(p.Payload)
	totalLen := ihl + payloadLen

	buf := make([]byte, totalLen)

	// Version and IHL
	buf[0] = (4 << 4) | (uint8(ihl/4) & 0x0F)
	buf[1] = p.TOS
	binary.BigEndian.PutUint16(buf[2:4], uint16(totalLen))
	binary.BigEndian.PutUint16(buf[4:6], p.ID)

	// Flags and Fragment Offset
	flagsFragment := (uint16(p.Flags) << 13) | (p.FragmentOffset & 0x1FFF)
	binary.BigEndian.PutUint16(buf[6:8], flagsFragment)

	buf[8] = p.TTL
	buf[9] = p.Protocol
	// Checksum at [10:12] - will be calculated
	copy(buf[12:16], p.SrcIP.To4())
	copy(buf[16:20], p.DstIP.To4())

	// Options
	if len(p.Options) > 0 {
		copy(buf[20:], p.Options)
	}

	// Calculate and set checksum
	checksum := Checksum(buf[:ihl])
	binary.BigEndian.PutUint16(buf[10:12], checksum)

	// Add payload
	copy(buf[ihl:], p.Payload)
	return buf
}

// Checksum calculates the IPv4 header checksum
func Checksum(data []byte) uint16 {
	sum := uint32(0)

	// Sum all 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Handle odd length
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

// String returns a string representation of the packet
func (p *Packet) String() string {
	protoName := protocolToString(p.Protocol)
	return fmt.Sprintf("IPv4: Src=%s Dst=%s Proto=%d(%s) TTL=%d Len=%d ID=%d",
		p.SrcIP, p.DstIP, p.Protocol, protoName, p.TTL, p.TotalLength, p.ID)
}

// protocolToString converts protocol number to string
func protocolToString(proto uint8) string {
	switch proto {
	case ProtocolICMP:
		return "ICMP"
	case ProtocolTCP:
		return "TCP"
	case ProtocolUDP:
		return "UDP"
	case ProtocolIPv6:
		return "IPv6"
	default:
		return fmt.Sprintf("Unknown(%d)", proto)
	}
}

// IsFragmented checks if the packet is fragmented
func (p *Packet) IsFragmented() bool {
	return p.FragmentOffset != 0 || p.Flags&FlagMF != 0
}

// PseudoHeader computes the pseudo-header for checksum calculation
func (p *Packet) PseudoHeader() []byte {
	buf := make([]byte, 12)
	copy(buf[0:4], p.SrcIP.To4())
	copy(buf[4:8], p.DstIP.To4())
	buf[8] = 0
	buf[9] = p.Protocol
	binary.BigEndian.PutUint16(buf[10:12], uint16(len(p.Payload)))
	return buf
}
```

#### Key Learning Points

- **IHL (Internet Header Length)**: Header length in 4-byte units (usually 5 = 20 bytes)
- **TTL**: Loop prevention. Decremented at each router, discarded when 0
- **Checksum**: Detects header corruption. The implementation computes it on send and validates it on receive, together with IHL and total length

---

### ✅ Verification Checkpoints

- [ ] The checksum function handles the Internet checksum.
- [ ] The wire IHL field represents header length in four-byte words.

---

### STEP 4: ICMP Message (L4) - Ping Implementation

#### Goal

ICMP (Internet Control Message Protocol) is used for network diagnostics. The most common use is "Ping" (Echo Request/Reply).

#### ICMP Echo Message Structure

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Type | Code | Checksum |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Identifier | Sequence Number |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Data |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

#### Implementation: ICMP Message (`pkg/icmp/message.go`)

```go
package icmp

import (
	"encoding/binary"
	"fmt"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
)

const (
	// Type values
	EchoReply              = 0
	DestinationUnreachable = 3
	SourceQuench           = 4
	Redirect               = 5
	EchoRequest            = 8
	TimeExceeded           = 11
	ParameterProblem       = 12
	Timestamp              = 13
	TimestampReply         = 14

	// DestinationUnreachable codes
	NetUnreachable      = 0
	HostUnreachable     = 1
	ProtocolUnreachable = 2
	PortUnreachable     = 3
	FragmentationNeeded = 4
	SourceRouteFailed   = 5

	// TimeExceeded codes
	TTLExceeded                    = 0
	FragmentReassemblyTimeExceeded = 1
)

// Message represents an ICMP message
type Message struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	// Type-specific fields
	ID   uint16
	Seq  uint16
	Data []byte
	// For error messages
	OriginalPacket []byte
}

// Parse parses an ICMP message from bytes
func Parse(data []byte) (*Message, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("message too short: %d < 8", len(data))
	}
	if ipv4.Checksum(data) != 0 {
		return nil, fmt.Errorf("invalid ICMP checksum")
	}

	msg := &Message{
		Type:     data[0],
		Code:     data[1],
		Checksum: binary.BigEndian.Uint16(data[2:4]),
		ID:       binary.BigEndian.Uint16(data[4:6]),
		Seq:      binary.BigEndian.Uint16(data[6:8]),
	}

	if len(data) > 8 {
		msg.Data = data[8:]
		if msg.IsError() {
			msg.OriginalPacket = data[8:]
		}
	}

	return msg, nil
}

// Marshal converts the message to bytes
func (m *Message) Marshal() []byte {
	// Minimum size is 8 bytes
	minLen := 8
	if m.Type == DestinationUnreachable || m.Type == TimeExceeded || m.Type == ParameterProblem {
		// Error messages have the original IP header + 8 bytes
		minLen = 8 + len(m.OriginalPacket)
	} else {
		minLen = 8 + len(m.Data)
	}

	buf := make([]byte, minLen)
	buf[0] = m.Type
	buf[1] = m.Code
	// Checksum at [2:4] - will be calculated
	binary.BigEndian.PutUint16(buf[4:6], m.ID)
	binary.BigEndian.PutUint16(buf[6:8], m.Seq)

	if len(m.Data) > 0 {
		copy(buf[8:], m.Data)
	} else if len(m.OriginalPacket) > 0 {
		copy(buf[8:], m.OriginalPacket)
	}

	// Calculate checksum (excluding checksum field itself)
	checksum := ipv4.Checksum(buf)
	binary.BigEndian.PutUint16(buf[2:4], checksum)

	return buf
}

// NewEchoRequest creates a new ICMP Echo Request
func NewEchoRequest(id, seq uint16, data []byte) *Message {
	return &Message{
		Type: EchoRequest,
		Code: 0,
		ID:   id,
		Seq:  seq,
		Data: data,
	}
}

// NewEchoReply creates a new ICMP Echo Reply
func NewEchoReply(id, seq uint16, data []byte) *Message {
	return &Message{
		Type: EchoReply,
		Code: 0,
		ID:   id,
		Seq:  seq,
		Data: data,
	}
}

// NewDestinationUnreachable creates a new Destination Unreachable message
func NewDestinationUnreachable(code uint8, originalPacket []byte) *Message {
	// Include original IP header + first 8 bytes
	dataLen := quotedPacketLength(originalPacket)

	return &Message{
		Type:           DestinationUnreachable,
		Code:           code,
		OriginalPacket: originalPacket[:dataLen],
	}
}

// NewTimeExceeded creates a new Time Exceeded message
func NewTimeExceeded(code uint8, originalPacket []byte) *Message {
	dataLen := quotedPacketLength(originalPacket)

	return &Message{
		Type:           TimeExceeded,
		Code:           code,
		OriginalPacket: originalPacket[:dataLen],
	}
}

// ICMP errors quote the original IPv4 header (including options) and up to
// eight payload bytes. Keep truncated inputs bounded by the supplied buffer.
func quotedPacketLength(packet []byte) int {
	if len(packet) < ipv4.HeaderSize {
		return len(packet)
	}
	headerLength := int(packet[0]&0x0f) * 4
	if headerLength < ipv4.HeaderSize {
		headerLength = ipv4.HeaderSize
	}
	return min(len(packet), headerLength+8)
}

// IsEchoRequest checks if this is an Echo Request
func (m *Message) IsEchoRequest() bool {
	return m.Type == EchoRequest
}

// IsEchoReply checks if this is an Echo Reply
func (m *Message) IsEchoReply() bool {
	return m.Type == EchoReply
}

// IsError checks if this is an error message
func (m *Message) IsError() bool {
	return m.Type == DestinationUnreachable ||
		m.Type == TimeExceeded ||
		m.Type == Redirect ||
		m.Type == SourceQuench ||
		m.Type == ParameterProblem
}

// String returns a string representation of the message
func (m *Message) String() string {
	typeName := typeToString(m.Type)
	if m.IsEchoRequest() || m.IsEchoReply() {
		return fmt.Sprintf("ICMP: Type=%d(%s) ID=%d Seq=%d DataLen=%d",
			m.Type, typeName, m.ID, m.Seq, len(m.Data))
	}
	return fmt.Sprintf("ICMP: Type=%d(%s) Code=%d", m.Type, typeName, m.Code)
}

// typeToString converts ICMP type to string
func typeToString(t uint8) string {
	switch t {
	case EchoReply:
		return "Echo Reply"
	case DestinationUnreachable:
		return "Dest Unreachable"
	case SourceQuench:
		return "Source Quench"
	case Redirect:
		return "Redirect"
	case EchoRequest:
		return "Echo Request"
	case TimeExceeded:
		return "Time Exceeded"
	case ParameterProblem:
		return "Parameter Problem"
	case Timestamp:
		return "Timestamp"
	case TimestampReply:
		return "Timestamp Reply"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

// ValidateChecksum validates the ICMP checksum
func (m *Message) ValidateChecksum() bool {
	data := m.Marshal()
	// Marshal computes a fresh checksum; validation must use the received one.
	binary.BigEndian.PutUint16(data[2:4], m.Checksum)
	calculated := ipv4.Checksum(data)
	return calculated == 0
}
```

#### Key Learning Points

- **Type 8/0**: Echo Request (8) triggers Echo Reply (0)
- **ID/Seq**: Identifiers to distinguish multiple Pings
- **Data**: Stores timestamps for RTT measurement

---

### STEP 5: Stack Integration (`main.go`) - Connecting Everything

#### Goal

Integrate all components to build a server that responds to Ping requests.

#### Implementation: Main Program (`main.go`)

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

### STEP 6: UDP Message (L4) — Connectionless Communication

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
+----------------------+----------------------+
| Source Port (16 bits) | Destination Port (16) |
+----------------------+----------------------+
| Length (16 bits)     | Checksum (16 bits)    |
+----------------------+----------------------+
| Data                                        |
+---------------------------------------------+
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
```

### 4. Observe Packets

```bash
sudo tcpdump -i veth-host -nn -vv icmp
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
PY

sudo tcpdump -i veth-host -nn -vv -XX 'udp port 7'
```

---

### 6. Observe and Corrupt UDP

Use `tcpdump -XX` to map the eight-byte UDP header and payload to the captured bytes. Then run the `pkg/udp` tests: flipping a checksum bit must make `VerifyChecksum` reject the datagram. Verify both accepted and rejected messages.

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

```bash
sudo ip link delete veth-host
sudo ip netns delete workshop
```

---

## 💡 Deep Dive Topics

1. **Zero-Copy**: Using `unsafe.Pointer` in CGO to avoid memory copies
2. **Checksum**: RFC 1071 ones' complement sum algorithm
3. **Endianness**: Importance of network byte order (Big Endian)
4. **Extensibility**: Design considerations for adding TCP/UDP

## 📚 References

- [RFC 791 (IPv4)](https://datatracker.ietf.org/doc/html/rfc791)
- [RFC 792 (ICMP)](https://datatracker.ietf.org/doc/html/rfc792)
- [RFC 1071 (Checksum)](https://datatracker.ietf.org/doc/html/rfc1071)

---

## 🔧 Troubleshooting

### Cannot Open a Raw Socket

Run with root privileges or grant `CAP_NET_RAW`, and specify both the interface and userspace IP.

```bash
sudo ./tcpip_stack -iface veth-host -ip 192.168.100.1
# Alternative to running as root:
getcap ./tcpip_stack
sudo setcap cap_net_raw+ep ./tcpip_stack
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

### Build Errors

The raw socket adapter requires Linux, CGO, and a C compiler.

```bash
gcc --version
go env CGO_ENABLED
sudo apt install build-essential linux-libc-dev
```

## 💻 Environment Notes

### For macOS Users

The raw socket application requires Linux `AF_PACKET`, which macOS does not provide. Run it in a Linux VM or Linux container with the required network capabilities. Pure packet parsing tests can run on macOS.

### For Windows Users

The raw socket application requires Linux. Use Ubuntu on WSL2 or a Linux VM. Pure Go packet parsing tests do not require raw sockets.

### For WSL2 Users

Check your Linux environment and the required networking features:

```bash
uname -r
```

If raw sockets or namespace setup are unavailable in your environment, use a Linux VM.
