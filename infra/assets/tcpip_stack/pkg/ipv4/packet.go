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
	FlagDF = 0x01 // Don't Fragment
	FlagMF = 0x02 // More Fragments
)

// Packet represents an IPv4 packet
type Packet struct {
	Version        uint8
	IHL            uint8  // Internet Header Length in 32-bit words
	TOS            uint8  // Type of Service
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
	if len(data) < int(ihl) {
		return nil, fmt.Errorf("packet too short for IHL: %d < %d", len(data), ihl)
	}

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
		Payload:        data[ihl:],
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
		ihl = HeaderSize + len(p.Options)
	}

	payloadLen := len(p.Payload)
	totalLen := ihl + payloadLen

	buf := make([]byte, ihl)

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
	result := append(buf, p.Payload...)

	return result
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
		sum = (sum & 0xFFFF) + (sum>>16)
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
