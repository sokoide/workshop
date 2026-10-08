# Userspace TCP/IP Stack

A from-scratch userspace network stack — Ethernet framing, IPv4, ICMP, and UDP — speaking directly over a raw socket, without the kernel's IP layer. Part of the [TCP/IP Stack Workshop](../../tcpip_stack_en.md) and the [UDP follow-up workshop](../../tcpip_stack_udp_en.md).

> **⚠️ Linux + root required**: the main program is guarded by `//go:build linux` and opens a raw socket, so it needs `sudo` on Linux. `pkg/rawsock` uses cgo, so `CGO_ENABLED=1` and a C toolchain are required to build.

## What this is

`main.go` builds a `Stack` bound to a network interface and an IP address that the kernel deliberately does not own, then processes frames in a loop: parse Ethernet, handle ARP replies, answer ICMP echo requests, and parse UDP packets. `cmd/ping` is a custom ping built on the same `pkg/` layers. The `setup-env` target creates an isolated network namespace with a veth pair so the stack can be tested safely against a real peer.

## Requirements

- Linux with root/sudo (raw sockets)
- Go 1.25.7 (`go.mod`), CGO enabled (cgo raw-socket wrapper in `pkg/rawsock`)
- Makefile targets: `build`, `run`, `ping`, `test`, `setup-env`, `run-veth`, `ping-test`, `teardown-env`, `interfaces`, `build-all`, `fmt`, `lint`

## How to run

The simplest path is the prepared veth environment (no changes to your real network):

```bash
make setup-env    # creates netns "workshop" + veth pair (192.168.100.0/24)
make run-veth     # sudo ./tcpip_stack -iface veth-host -ip 192.168.100.1
make ping-test    # ping the userspace stack from inside the namespace
make teardown-env # cleanup
```

Other entry points:

```bash
make interfaces   # list available interfaces
make run          # run on a physical interface (requires sudo)
make ping         # build and run the custom ping (e.g. -iface eth0 -target <ip>)
make test         # unit tests for each protocol layer
```

See the [TCP/IP stack workshop](../../tcpip_stack_en.md) and the [UDP workshop](../../tcpip_stack_udp_en.md) for packet-level walkthroughs of each layer.

## Layout

```text
.
├── main.go            # Stack: frame loop, ARP/ICMP handling (linux-only)
├── cmd/ping/          # Custom ping built on the stack
├── pkg/
│   ├── rawsock/       # cgo raw socket wrapper (AF_PACKET)
│   ├── ethernet/      # Ethernet frame parse/marshal
│   ├── ipv4/          # IPv4 header + checksum
│   ├── icmp/          # ICMP echo (ping) support
│   └── udp/           # UDP parse/verify/reply
└── Makefile           # build / run / ping / setup-env / test targets
```
