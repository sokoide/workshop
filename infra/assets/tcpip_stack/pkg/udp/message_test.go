package udp

import (
	"net"
	"testing"
)

var (
	testSrcIP = net.IPv4(192, 0, 2, 10)
	testDstIP = net.IPv4(192, 0, 2, 20)
)

func TestMarshalParseAndVerifyChecksum(t *testing.T) {
	message := &Message{SrcPort: 49152, DstPort: EchoPort, Data: []byte("hello")}
	datagram, err := message.Marshal(testSrcIP, testDstIP)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	parsed, err := Parse(datagram)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if parsed.Length != 13 || string(parsed.Data) != "hello" {
		t.Fatalf("Parse() = %+v, want length 13 and payload hello", parsed)
	}
	if err := VerifyChecksum(datagram, testSrcIP, testDstIP); err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
}

func TestParseRejectsInvalidLength(t *testing.T) {
	datagram := []byte{0, 1, 0, 7, 0, 8, 0, 0, 'x'}
	if _, err := Parse(datagram); err == nil {
		t.Fatal("Parse() accepted a datagram whose length excludes trailing data")
	}
}

func TestVerifyChecksumRejectsCorruption(t *testing.T) {
	datagram, err := (&Message{SrcPort: 49152, DstPort: EchoPort, Data: []byte("hello")}).Marshal(testSrcIP, testDstIP)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	datagram[len(datagram)-1] ^= 0x01
	if err := VerifyChecksum(datagram, testSrcIP, testDstIP); err == nil {
		t.Fatal("VerifyChecksum() accepted a one-bit-corrupted datagram")
	}
}
