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
