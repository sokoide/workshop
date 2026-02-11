package icmp

import (
	"encoding/binary"
	"fmt"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
)

const (
	// Type values
	EchoReply     = 0
	DestinationUnreachable = 3
	SourceQuench   = 4
	Redirect       = 5
	EchoRequest    = 8
	TimeExceeded   = 11
	ParameterProblem = 12
	Timestamp      = 13
	TimestampReply = 14

	// DestinationUnreachable codes
	NetUnreachable = 0
	HostUnreachable = 1
	ProtocolUnreachable = 2
	PortUnreachable = 3
	FragmentationNeeded = 4
	SourceRouteFailed = 5

	// TimeExceeded codes
	TTLExceeded = 0
	FragmentReassemblyTimeExceeded = 1
)

// Message represents an ICMP message
type Message struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	// Type-specific fields
	ID       uint16
	Seq      uint16
	Data     []byte
	// For error messages
	OriginalPacket []byte
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
	dataLen := len(originalPacket)
	if dataLen > 8 {
		dataLen = 8
	}

	return &Message{
		Type:          DestinationUnreachable,
		Code:          code,
		OriginalPacket: originalPacket[:dataLen],
	}
}

// NewTimeExceeded creates a new Time Exceeded message
func NewTimeExceeded(code uint8, originalPacket []byte) *Message {
	dataLen := len(originalPacket)
	if dataLen > 8 {
		dataLen = 8
	}

	return &Message{
		Type:          TimeExceeded,
		Code:          code,
		OriginalPacket: originalPacket[:dataLen],
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
	calculated := ipv4.Checksum(data)
	return calculated == 0
}
