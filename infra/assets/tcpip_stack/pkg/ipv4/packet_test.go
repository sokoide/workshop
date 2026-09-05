package ipv4

import (
	"encoding/binary"
	"net"
	"testing"
)

func packetBytes() []byte {
	return (&Packet{SrcIP: net.IPv4(192, 0, 2, 1), DstIP: net.IPv4(192, 0, 2, 2), TTL: 64, Protocol: ProtocolUDP, Payload: []byte("test")}).Marshal()
}

func TestRejectInvalidIHL(t *testing.T) {
	for words := byte(0); words < 5; words++ {
		data := packetBytes()
		data[0] = 0x40 | words
		if _, err := Parse(data); err == nil {
			t.Errorf("accepted IHL=%d", words)
		}
	}
}

func TestRejectCorruptHeader(t *testing.T) {
	data := packetBytes()
	data[8] ^= 1
	if _, err := Parse(data); err == nil {
		t.Fatal("accepted corrupted IPv4 header")
	}
}

func TestFragmentFlagsMatchWire(t *testing.T) {
	for _, tc := range []struct {
		wire       uint16
		fragmented bool
	}{{0x2000, true}, {0x4000, false}, {1, true}, {0, false}} {
		data := packetBytes()
		binary.BigEndian.PutUint16(data[6:8], tc.wire)
		data[10], data[11] = 0, 0
		binary.BigEndian.PutUint16(data[10:12], Checksum(data[:HeaderSize]))
		packet, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if packet.IsFragmented() != tc.fragmented {
			t.Errorf("flags=%04x: fragmented=%v", tc.wire, packet.IsFragmented())
		}
	}
}

func TestOptionsArePaddedToWordBoundary(t *testing.T) {
	packet := &Packet{TTL: 64, Options: []byte{1}, Payload: []byte("payload")}
	data := packet.Marshal()
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.IHL != 24 || string(got.Payload) != "payload" {
		t.Fatalf("IHL=%d payload=%q", got.IHL, got.Payload)
	}
}
