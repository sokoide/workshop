# TCP/IP Protocol Stack Workshop: Building the "Soul of Communication" from Scratch

In this workshop, you will combine Go and C to **build a TCP/IP protocol stack from the ground up**. By implementing the "packet generation and decomposition" that OS kernels usually hide, you will understand network principles not as abstract concepts, but as concrete data structures.

---

## 🏗 Architecture: Russian Doll (Encapsulation)

The essence of network communication is "Encapsulation." Higher-layer data is wrapped as the "payload" of lower layers.

```
+----------------------------------------------------+
| Ethernet Frame (L2)                                |
| +------------------------------------------------+ |
| | IPv4 Packet (L3)                               | |
| | +--------------------------------------------+ | |
| | | ICMP Message (L4)                          | | |
| | | +----------------------------------------+ | | |
| | | | Payload (Data)                         | | | |
| | | +----------------------------------------+ | | |
| | +--------------------------------------------+ | |
| +------------------------------------------------+ |
+----------------------------------------------------+
```

### Components to Implement

| Layer | Component | File | Description |
|-------|-----------|------|-------------|
| L1 | Raw Socket | `pkg/rawsock/` | Bypass kernel, communicate directly with NIC |
| L2 | Ethernet Frame | `pkg/ethernet/frame.go` | MAC address-based frame handling |
| L3 | IPv4 Packet | `pkg/ipv4/packet.go` | IP address-based packet handling |
| L4 | ICMP Message | `pkg/icmp/message.go` | Ping (Echo Request/Reply) implementation |
| App | Stack | `main.go` | Server integrating all components |

---

## 🛠 Environment Setup

This workshop requires **Linux (Ubuntu 22.04/24.04 recommended)** as it involves low-level Raw Socket operations.

### 1. Required Tools

```bash
sudo apt update && sudo apt install -y golang-go gcc tcpdump wireshark iproute2
```

### 2. Project Structure

```bash
mkdir -p tcpip_stack/pkg/{rawsock,ethernet,ipv4,icmp}
mkdir -p tcpip_stack/cmd/ping
cd tcpip_stack
go mod init tcpip_stack
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

# 5. Assign IP addresses. Set .1 for host side and .2 for namespace side
sudo ip addr add 192.168.100.1/24 dev veth-host
sudo ip netns exec workshop ip addr add 192.168.100.2/24 dev veth-ns
```

---

## 🚀 Implementation Steps

### STEP 1: Raw Socket (L1) - Bypassing the Kernel

#### Goal

Normal applications use `TCP/UDP Sockets`, but the OS automatically generates L2/L3 headers for them. In our custom stack, we use **AF_PACKET** to read/write "raw data" directly from the NIC.

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

```
Regular Socket (AF_INET)
+-----------+     +------------------------+
|   App     | <-- | Kernel handles TCP/IP  |
+-----------+     +------------------------+

Raw Socket (AF_PACKET)
+-----------+     +------------------------------------+
|   App     | <-- | Raw Ethernet frame (bytes)         |
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

| Argument | Value | Meaning |
|----------|-------|---------|
| Domain | `AF_PACKET` | Packet-level socket (L2) |
| Type | `SOCK_RAW` | Get raw packets including headers |
| Protocol | `ETH_P_ALL` (0x0003) | Receive all protocol types |

##### Other Protocol Values

| Protocol | Value | Description |
|----------|-------|-------------|
| `ETH_P_ALL` | 0x0003 | Receive all packets |
| `ETH_P_IP` | 0x0800 | Receive IPv4 only |
| `ETH_P_IPV6` | 0x86DD | Receive IPv6 only |
| `ETH_P_ARP` | 0x0806 | Receive ARP only |

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

### STEP 2: Ethernet Frame (L2) - MAC Address-Based Communication

#### Goal

Ethernet controls communication with physically adjacent devices using MAC addresses (48-bit) to identify communication partners within the same segment.

#### Ethernet II Frame Structure

```
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

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Version|  IHL  |    TOS       |          Total Length         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Identification        |Flags|     Fragment Offset     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  TTL  | Protocol|        Header Checksum                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Source Address                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Destination Address                        |
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
	// Minimum header size (no options)
	HeaderSize = 20

	// Maximum packet size
	MaxPacketSize = 65535

	// Protocol numbers
	ProtocolICMP = 1
	ProtocolTCP  = 6
	ProtocolUDP  = 17

	// Flags
	FlagDF = 0x01 // Don't Fragment
	FlagMF = 0x02 // More Fragments
)

// Packet represents an IPv4 packet
type Packet struct {
	Version        uint8  // Always 4
	IHL            uint8  // Internet Header Length (4-byte units)
	TOS            uint8  // Type of Service
	TotalLength    uint16 // Header + Payload
	ID             uint16 // Identifier (for fragment reassembly)
	Flags          uint8
	FragmentOffset uint16 // Fragment offset (8-byte units)
	TTL            uint8  // Time to Live (hop limit)
	Protocol       uint8  // Upper layer protocol
	Checksum       uint16 // Header checksum
	SrcIP          net.IP
	DstIP          net.IP
	Options        []byte // IP options (usually not used)
	Payload        []byte
}

// Parse parses an IPv4 packet from bytes
func Parse(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("packet too short: %d < %d", len(data), HeaderSize)
	}

	// Version check
	version := data[0] >> 4
	if version != 4 {
		return nil, fmt.Errorf("not IPv4: version=%d", version)
	}

	// IHL (Internet Header Length) is in 4-byte units
	ihl := (data[0] & 0x0F) * 4
	if len(data) < int(ihl) {
		return nil, fmt.Errorf("packet too short for IHL: %d < %d", len(data), ihl)
	}

	// Flags and Fragment Offset are packed into 16 bits
	//   Flags: upper 3 bits
	//   Fragment Offset: lower 13 bits
	flagsFragment := binary.BigEndian.Uint16(data[6:8])

	pkt := &Packet{
		Version:        version,
		IHL:            ihl,
		TOS:            data[1],
		TotalLength:    binary.BigEndian.Uint16(data[2:4]),
		ID:             binary.BigEndian.Uint16(data[4:6]),
		Flags:          uint8((flagsFragment >> 13) & 0x07),
		FragmentOffset: flagsFragment & 0x1FFF,
		TTL:            data[8],
		Protocol:       data[9],
		Checksum:       binary.BigEndian.Uint16(data[10:12]),
		SrcIP:          net.IP(data[12:16]),
		DstIP:          net.IP(data[16:20]),
		Payload:        data[ihl:], // Payload starts after IHL
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
		ihl = HeaderSize + len(p.Options)
	}

	payloadLen := len(p.Payload)
	totalLen := ihl + payloadLen

	buf := make([]byte, ihl)

	// Version (4 bits) + IHL (4 bits)
	buf[0] = (4 << 4) | (uint8(ihl/4) & 0x0F)
	buf[1] = p.TOS
	binary.BigEndian.PutUint16(buf[2:4], uint16(totalLen))
	binary.BigEndian.PutUint16(buf[4:6], p.ID)

	// Flags (3 bits) + Fragment Offset (13 bits)
	flagsFragment := (uint16(p.Flags) << 13) | (p.FragmentOffset & 0x1FFF)
	binary.BigEndian.PutUint16(buf[6:8], flagsFragment)

	buf[8] = p.TTL
	buf[9] = p.Protocol
	// Checksum will be calculated

	// IP addresses (convert to 4 bytes)
	copy(buf[12:16], p.SrcIP.To4())
	copy(buf[16:20], p.DstIP.To4())

	// Options
	if len(p.Options) > 0 {
		copy(buf[20:], p.Options)
	}

	// Calculate checksum
	checksum := Checksum(buf[:ihl])
	binary.BigEndian.PutUint16(buf[10:12], checksum)

	// Add payload
	result := append(buf, p.Payload...)

	return result
}

// Checksum calculates the IPv4 header checksum
// RFC 1071: 16-bit ones' complement sum of ones' complement
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

	// Fold 32-bit sum to 16 bits (add carry)
	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// Return ones' complement
	return ^uint16(sum)
}

// IsFragmented checks if the packet is fragmented
func (p *Packet) IsFragmented() bool {
	return p.FragmentOffset != 0 || p.Flags&FlagMF != 0
}

// PseudoHeader generates pseudo-header for TCP/UDP checksum calculation
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
- **Checksum**: Detects header corruption. Calculated on send, verified on receive

---

### STEP 4: ICMP Message (L4) - Ping Implementation

#### Goal

ICMP (Internet Control Message Protocol) is used for network diagnostics. The most common use is "Ping" (Echo Request/Reply).

#### ICMP Echo Message Structure

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |     Code      |          Checksum             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Identifier          |        Sequence Number        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                             Data                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

#### Implementation: ICMP Message (`pkg/icmp/message.go`)

```go
package icmp

import (
	"encoding/binary"
	"fmt"

	"tcpip_stack/pkg/ipv4"
)

const (
	// ICMP types
	EchoReply              = 0
	DestinationUnreachable = 3
	EchoRequest            = 8
	TimeExceeded           = 11
)

// Message represents an ICMP message
type Message struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16 // For Echo Request/Reply
	Seq      uint16 // For Echo Request/Reply
	Data     []byte
}

// Parse parses an ICMP message from bytes
func Parse(data []byte) (*Message, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("message too short: %d < 8", len(data))
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
	}

	return msg, nil
}

// Marshal converts the message to bytes
func (m *Message) Marshal() []byte {
	buf := make([]byte, 8+len(m.Data))

	buf[0] = m.Type
	buf[1] = m.Code
	// Checksum will be calculated
	binary.BigEndian.PutUint16(buf[4:6], m.ID)
	binary.BigEndian.PutUint16(buf[6:8], m.Seq)

	if len(m.Data) > 0 {
		copy(buf[8:], m.Data)
	}

	// Calculate checksum (same algorithm as IPv4)
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

// IsEchoRequest checks if this is an Echo Request
func (m *Message) IsEchoRequest() bool {
	return m.Type == EchoRequest
}

// IsEchoReply checks if this is an Echo Reply
func (m *Message) IsEchoReply() bool {
	return m.Type == EchoReply
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
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"tcpip_stack/pkg/ethernet"
	"tcpip_stack/pkg/icmp"
	"tcpip_stack/pkg/ipv4"
	"tcpip_stack/pkg/rawsock"
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
	// Open raw socket
	sock, err := rawsock.Open(iface)
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

	// Get interface info
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get interface: %w", err)
	}

	// Get IP address
	addrs, err := ifaceObj.Addrs()
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get addresses: %w", err)
	}

	var srcIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
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
			// Receive packet
			n, err := s.sock.Recv(buf)
			if err != nil {
				continue
			}

			// Copy bytes (will be overwritten by next receive)
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
	// L2: Parse Ethernet frame
	frame, err := ethernet.Parse(data)
	if err != nil {
		return err
	}

	// Ignore packets not addressed to us
	if !s.shouldProcessFrame(frame) {
		return nil
	}

	// Ignore non-IPv4
	if frame.EtherType != ethernet.TypeIPv4 {
		return nil
	}

	// L3: Parse IPv4 packet
	pkt, err := ipv4.Parse(frame.Payload)
	if err != nil {
		return err
	}

	// Ignore non-ICMP
	if pkt.Protocol != ipv4.ProtocolICMP {
		return nil
	}

	// L4: Parse ICMP message
	msg, err := icmp.Parse(pkt.Payload)
	if err != nil {
		return err
	}

	// Respond to Ping (Echo Request)
	if msg.IsEchoRequest() && pkt.DstIP.Equal(s.srcIP) {
		log.Printf("Ping from %s: ID=%d Seq=%d", pkt.SrcIP, msg.ID, msg.Seq)
		return s.sendEchoReply(pkt, msg, frame.SrcMAC)
	}

	return nil
}

// shouldProcessFrame checks if we should process this frame
func (s *Stack) shouldProcessFrame(frame *ethernet.Frame) bool {
	if frame.DstMAC.String() == s.srcMAC.String() {
		return true // Unicast
	}
	if ethernet.IsBroadcast(frame.DstMAC) {
		return true // Broadcast
	}
	if ethernet.IsMulticast(frame.DstMAC) {
		return true // Multicast
	}
	return false
}

// sendEchoReply sends an ICMP Echo Reply
func (s *Stack) sendEchoReply(reqPkt *ipv4.Packet, reqMsg *icmp.Message, dstMAC net.HardwareAddr) error {
	// L4: Create ICMP Echo Reply
	reply := icmp.NewEchoReply(reqMsg.ID, reqMsg.Seq, reqMsg.Data)
	icmpData := reply.Marshal()

	// L3: Create IPv4 packet
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

	// L2: Create Ethernet frame
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
sudo ./tcpip_stack -iface veth-host
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

---

## 📊 Packet Flow Diagram

```
=================== Receive Path (Rx) ===================

NIC --> Raw Socket --> Ethernet.Parse --> IPv4.Parse --> ICMP.Parse
                           |                  |               |
                      [MAC Filter]      [IP Filter]    [Type Check]
                           |                  |               |
                        Pass?              Pass?           Type=8?
                           |                  |               |
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

```
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

```
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

```
T1: Execute ping 192.168.100.1 from Namespace
    |
    v
T2: Packet arrives (Echo Request)
    +---------------------------------------------------+
    | Ethernet: Dst=FF:FF:FF:FF:FF:FF                   |
    | IPv4:     Src=192.168.100.2, Dst=192.168.100.1    |
    | ICMP:     Type=8, ID=1234, Seq=1                  |
    +---------------------------------------------------+
    |
    v  Arrives at veth-host
    |
T3: Custom stack receives
    |  -> MAC Filter: broadcast -> Pass
    |  -> IP Filter:  DstIP is ours -> Pass
    |  -> Type Check: Type=8 -> Generate Reply
    |
    v
T4: Packet sent (Echo Reply)
    +---------------------------------------------------+
    | Ethernet: Dst=xx:xx:xx:xx:xx:xx                   |
    | IPv4:     Src=192.168.100.1, Dst=192.168.100.2    |
    | ICMP:     Type=0, ID=1234, Seq=1                  |
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
