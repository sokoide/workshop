# TCP/IP Protocol Stack Workshop — Part 1: Ethernet, IP & ICMP

> **⏱️ Estimated Time**: Approx. 100 minutes (Part 1 of 2; the complete workshop takes approx. 180 minutes)
> **🎯 Level**: Advanced
> **💻 Environment**: Linux (root/sudo)

In this two-part workshop, you will combine Go and C to **build a TCP/IP protocol stack from the ground up**. By implementing the "packet generation and decomposition" that OS kernels usually hide, you will understand network principles not as abstract concepts, but as concrete data structures. Part 1 covers the observation step and the packet socket, Ethernet, IPv4, and ICMP layers; [Part 2: UDP & Stack Integration](tcpip_stack_udp_en.md) adds UDP datagrams, integrates everything into `main.go`, and runs the live ping and UDP echo tests.

> **💡 Glossary**: Please refer to [Raw Socket](glossary_en.md#protocol) or [Encapsulation](glossary_en.md#protocol) in the [Glossary](glossary_en.md) for technical terms used in this workshop.

## Goal

Observe what the kernel usually hides, then implement it yourself: by the end of this part you will have built the `pkg/rawsock`, `pkg/ethernet`, `pkg/ipv4`, and `pkg/icmp` packages on top of an AF_PACKET raw socket, and confirmed with `go build ./...` that the whole stack compiles. The integrated stack — live ping and UDP echo from inside the namespace — is verified in [Part 2](tcpip_stack_udp_en.md).

## Prerequisites

- [ ] Linux (Ubuntu 22.04/24.04 recommended) — Raw Sockets (`AF_PACKET`) and network namespaces are required
- [ ] Go 1.22 or later (`go version`)
- [ ] GCC with CGO enabled (`gcc --version`, `go env CGO_ENABLED`)
- [ ] `tcpdump` and `iproute2` (`ip`, `ip netns`) — `sudo apt install -y golang-go gcc tcpdump iproute2`
- [ ] Root/sudo privileges (creating network namespaces and opening raw sockets requires `CAP_NET_RAW`)

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

Part 1 implements the first four rows (Packet Socket through ICMP). UDP and the integrating `main.go` stack are covered in [Part 2](tcpip_stack_udp_en.md).

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
# Expected output: TCP packets with flags [S], [S.] (handshake), then [P.] data segments and [F]/[R] teardown
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

Read the [executable IPv4 implementation](assets/tcpip_stack/pkg/ipv4/packet.go) and its [tests](assets/tcpip_stack/pkg/ipv4/packet_test.go). `Parse` validates the version, IHL, total length, and header checksum before extracting the fields; `Marshal` builds the header, copies the payload, and computes the checksum. Use this source directly to avoid maintaining a second copy of the implementation in the tutorial.

Copy the reference implementation into your project (run from the workshop repository root, the parent of `tcpip_stack/`):

```bash
cp assets/tcpip_stack/pkg/ipv4/packet.go assets/tcpip_stack/pkg/ipv4/packet_test.go tcpip_stack/pkg/ipv4/
```

The heart of the module is the RFC 1071 ones' complement checksum:

```go
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
```

`PseudoHeader` builds the 12-byte prefix (source IP, destination IP, zero, protocol, length) that the transport-layer checksums in [Part 2](tcpip_stack_udp_en.md) cover.

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

Read the [executable ICMP implementation](assets/tcpip_stack/pkg/icmp/message.go) and its [tests](assets/tcpip_stack/pkg/icmp/message_test.go). `Parse` verifies the checksum before extracting Type/Code/ID/Seq, `NewEchoRequest`/`NewEchoReply` build the Ping pair, and the error constructors quote the original IPv4 header plus up to eight payload bytes. Use this source directly to avoid maintaining a second copy of the implementation in the tutorial.

Copy the reference implementation into your project (run from the workshop repository root):

```bash
cp assets/tcpip_stack/pkg/icmp/message.go assets/tcpip_stack/pkg/icmp/message_test.go tcpip_stack/pkg/icmp/
```

#### Key Learning Points

- **Type 8/0**: Echo Request (8) triggers Echo Reply (0)
- **ID/Seq**: Identifiers to distinguish multiple Pings
- **Data**: Stores timestamps for RTT measurement

## 🧪 Testing

Part 1 verifies that the packages you implemented compile. Building the integrated stack and running the ping and UDP echo tests is [Part 2](tcpip_stack_udp_en.md).

### Build All Packages

```bash
go build ./...
# Expected output: none — all packages compile
```

If you copied the asset test files while creating the packages, `go test ./...` runs them.

---

## What you learned

- How the kernel encapsulates application data into Ethernet (L2) and IPv4 (L3) frames, and how to parse each header from raw bytes.
- How to use Linux `AF_PACKET` raw sockets (via CGO) to observe and inject frames on a veth interface.
- How network namespaces and veth pairs provide an isolated sandbox for low-level networking experiments.
- How the RFC 1071 checksum and network byte order (Big Endian) protect packet integrity.

**Next step**: [Part 2: UDP & Stack Integration](tcpip_stack_udp_en.md) implements UDP datagrams, wires every component into `main.go`, and verifies the stack with live ping and UDP echo tests.

---

## 🧹 Cleanup

If you are continuing straight to [Part 2](tcpip_stack_udp_en.md), keep the sandbox — the next part reuses it. To remove it instead, run:

```bash
sudo ip link delete veth-host
sudo ip netns delete workshop
```

---

## 💡 Deep Dive Topics

1. **Zero-Copy**: Using `unsafe.Pointer` in CGO to avoid memory copies
2. **Checksum**: RFC 1071 ones' complement sum algorithm
3. **Endianness**: Importance of network byte order (Big Endian)

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
