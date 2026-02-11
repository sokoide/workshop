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
	fd     C.int
	iface  string
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

	return net.HardwareAddr{byte(mac[0]), byte(mac[1]), byte(mac[2]), byte(mac[3]), byte(mac[4]), byte(mac[5])}, nil
}

// GetMTU returns the MTU of the interface
func GetMTU(iface string) (int, error) {
	cIface := C.CString(iface)
	defer C.free(unsafe.Pointer(cIface))

	mtu := C.rawsock_get_mtu(cIface)
	if mtu < 0 {
		return 0, fmt.Errorf("failed to get MTU")
	}

	return int(mtu), nil
}

// SetPromisc enables or disables promiscuous mode
func (s *Socket) SetPromisc(enable bool) error {
	cIface := C.CString(s.iface)
	defer C.free(unsafe.Pointer(cIface))

	enableInt := 0
	if enable {
		enableInt = 1
	}

	ret := C.rawsock_set_promisc(s.fd, cIface, C.int(enableInt))
	if ret < 0 {
		return fmt.Errorf("failed to set promiscuous mode")
	}

	return nil
}
