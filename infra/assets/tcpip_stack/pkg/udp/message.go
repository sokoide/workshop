//go:build linux

// Package udp parses, validates, and marshals IPv4 UDP datagrams.
package udp

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
)

const (
	// HeaderSize is the fixed UDP header size.
	HeaderSize = 8

	// EchoPort is the UDP Echo Protocol port (RFC 862).
	EchoPort = 7
)

// Message represents a UDP datagram.
type Message struct {
	SrcPort  uint16
	DstPort  uint16
	Length   uint16
	Checksum uint16
	Data     []byte
}

// Parse parses one UDP datagram. The UDP length must describe the entire
// supplied IPv4 payload; accepting trailing bytes would hide a framing error.
func Parse(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("datagram too short: %d < %d", len(data), HeaderSize)
	}

	length := int(binary.BigEndian.Uint16(data[4:6]))
	if length < HeaderSize || length != len(data) {
		return nil, fmt.Errorf("invalid UDP length: declared=%d actual=%d", length, len(data))
	}

	return &Message{
		SrcPort:  binary.BigEndian.Uint16(data[0:2]),
		DstPort:  binary.BigEndian.Uint16(data[2:4]),
		Length:   uint16(length),
		Checksum: binary.BigEndian.Uint16(data[6:8]),
		Data:     data[HeaderSize:],
	}, nil
}

// VerifyChecksum verifies an IPv4 UDP checksum. A zero checksum means that
// the sender omitted checksum generation, which IPv4 permits.
func VerifyChecksum(data []byte, srcIP, dstIP net.IP) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("datagram too short: %d < %d", len(data), HeaderSize)
	}
	if binary.BigEndian.Uint16(data[6:8]) == 0 {
		return nil
	}

	pseudo, err := pseudoHeader(srcIP, dstIP, len(data))
	if err != nil {
		return err
	}
	if ipv4.Checksum(append(pseudo, data...)) != 0 {
		return fmt.Errorf("invalid UDP checksum")
	}
	return nil
}

// Marshal converts the message into an IPv4 UDP datagram with a checksum.
func (m *Message) Marshal(srcIP, dstIP net.IP) ([]byte, error) {
	length := HeaderSize + len(m.Data)
	if length > 0xffff {
		return nil, fmt.Errorf("datagram too large: %d > 65535", length)
	}

	buf := make([]byte, length)
	binary.BigEndian.PutUint16(buf[0:2], m.SrcPort)
	binary.BigEndian.PutUint16(buf[2:4], m.DstPort)
	binary.BigEndian.PutUint16(buf[4:6], uint16(length))
	copy(buf[HeaderSize:], m.Data)

	pseudo, err := pseudoHeader(srcIP, dstIP, length)
	if err != nil {
		return nil, err
	}
	checksum := ipv4.Checksum(append(pseudo, buf...))
	if checksum == 0 {
		checksum = 0xffff
	}
	binary.BigEndian.PutUint16(buf[6:8], checksum)

	return buf, nil
}

func pseudoHeader(srcIP, dstIP net.IP, length int) ([]byte, error) {
	src := srcIP.To4()
	dst := dstIP.To4()
	if src == nil || dst == nil {
		return nil, fmt.Errorf("UDP over IPv4 requires IPv4 addresses")
	}

	buf := make([]byte, 12)
	copy(buf[0:4], src)
	copy(buf[4:8], dst)
	buf[9] = ipv4.ProtocolUDP
	binary.BigEndian.PutUint16(buf[10:12], uint16(length))
	return buf, nil
}
