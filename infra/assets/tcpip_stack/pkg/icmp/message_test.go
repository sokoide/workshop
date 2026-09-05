package icmp

import "testing"

func TestChecksumRejectsCorruption(t *testing.T) {
	data := NewEchoRequest(1, 2, []byte("payload")).Marshal()
	parsed, err := Parse(data)
	if err != nil || !parsed.ValidateChecksum() {
		t.Fatalf("valid message: %v", err)
	}
	parsed.Checksum ^= 1
	if parsed.ValidateChecksum() {
		t.Fatal("accepted corrupt stored checksum")
	}
	data[len(data)-1] ^= 1
	if _, err := Parse(data); err == nil {
		t.Fatal("accepted corrupt received message")
	}
}

func TestErrorIncludesOriginalHeaderAndEightBytes(t *testing.T) {
	original := make([]byte, 40)
	original[0] = 0x46 // 24-byte header with options
	for _, msg := range []*Message{NewDestinationUnreachable(PortUnreachable, original), NewTimeExceeded(TTLExceeded, original)} {
		wire := msg.Marshal()
		if len(wire) != 8+24+8 {
			t.Fatalf("quoted packet truncated: %d", len(wire))
		}
		parsed, err := Parse(wire)
		if err != nil || !parsed.ValidateChecksum() {
			t.Fatalf("error message round trip: %v", err)
		}
	}
}
