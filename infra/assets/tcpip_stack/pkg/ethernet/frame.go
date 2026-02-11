package ethernet

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// HeaderSize is the size of Ethernet II header
	HeaderSize = 14

	// MinFrameSize is the minimum Ethernet frame size (with padding)
	MinFrameSize = 60

	// MaxFrameSize is the standard maximum Ethernet frame size
	MaxFrameSize = 1514

	// EtherType values
	TypeIPv4  = 0x0800
	TypeARP   = 0x0806
	TypeIPv6  = 0x86DD
	TypeVLAN  = 0x8100
	TypeQinQ  = 0x88A8
	TypeMPLS  = 0x8847
	TypeLLDP  = 0x88CC
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
		DstMAC:    net.HardwareAddr(data[0:6]),
		SrcMAC:    net.HardwareAddr(data[6:12]),
		EtherType: binary.BigEndian.Uint16(data[12:14]),
		Payload:   data[14:],
	}, nil
}

// Marshal converts the frame to bytes
func (f *Frame) Marshal() []byte {
	// Calculate total size
	payloadLen := len(f.Payload)
	totalLen := HeaderSize + payloadLen

	buf := make([]byte, totalLen)

	// Copy MAC addresses
	copy(buf[0:6], f.DstMAC)
	copy(buf[6:12], f.SrcMAC)

	// Set EtherType
	binary.BigEndian.PutUint16(buf[12:14], f.EtherType)

	// Copy payload
	if payloadLen > 0 {
		copy(buf[14:], f.Payload)
	}

	// Pad to minimum frame size if needed
	if totalLen < MinFrameSize {
		padded := make([]byte, MinFrameSize)
		copy(padded, buf)
		return padded
	}

	return buf
}

// String returns a string representation of the frame
func (f *Frame) String() string {
	etherTypeName := etherTypeToString(f.EtherType)
	return fmt.Sprintf("Ethernet: Dst=%s Src=%s Type=0x%04x(%s) PayloadLen=%d",
		f.DstMAC, f.SrcMAC, f.EtherType, etherTypeName, len(f.Payload))
}

// etherTypeToString converts EtherType to string
func etherTypeToString(etype uint16) string {
	switch etype {
	case TypeIPv4:
		return "IPv4"
	case TypeARP:
		return "ARP"
	case TypeIPv6:
		return "IPv6"
	case TypeVLAN:
		return "VLAN"
	case TypeQinQ:
		return "QinQ"
	case TypeMPLS:
		return "MPLS"
	case TypeLLDP:
		return "LLDP"
	default:
		return "Unknown"
	}
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
func IsMulticast(mac net.HardwareAddr) bool {
	if len(mac) != 6 {
		return false
	}
	return mac[0]&0x01 == 0x01
}
