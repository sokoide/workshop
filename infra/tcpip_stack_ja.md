# TCP/IP プロトコルスタック実習：スクラッチから構築する「通信の正体」

この実習では、Go と C を組み合わせて **TCP/IP プロトコルスタックをゼロから構築** します。OS カーネルが隠蔽している「パケットの生成・分解」を自らの手で実装することで、ネットワークの動作原理を抽象概念ではなく、具体的なデータ構造として理解することを目標とします。

---

## 🏗 アーキテクチャ：ロシアのマトリョーシカ（カプセル化）

ネットワーク通信の本質は「カプセル化」です。上位レイヤーのデータが下位レイヤーの「ペイロード」として包まれていきます。

```
+----------------------------------------------------+
| Ethernet Frame (L2)                                |
| +------------------------------------------------+ |
| | IPv4 Packet (L3)                               | |
| | +--------------------------------------------+ | |
| | | ICMP Message (L4)                          | | |
| | | +----------------------------------------+ | | |
| | | | Payload (Data)                         | | | |
| | | +----------------------------------------+ | | |
| | +--------------------------------------------+ | |
| +------------------------------------------------+ |
+----------------------------------------------------+
```

### 今回実装するコンポーネント

| レイヤー | コンポーネント | ファイル | 実装内容 |
|---------|--------------|---------|---------|
| L1 | Raw Socket | `pkg/rawsock/` | カーネルをバイパスしてNICと直接通信 |
| L2 | Ethernet Frame | `pkg/ethernet/frame.go` | MAC アドレスベースのフレーム処理 |
| L3 | IPv4 Packet | `pkg/ipv4/packet.go` | IP アドレスベースのパケット処理 |
| L4 | ICMP Message | `pkg/icmp/message.go` | Ping (Echo Request/Reply) の実装 |
| App | Stack | `main.go` | 全体を統合するサーバー |

---

## 🛠 開発環境の準備

本実習は、低レベルな Raw Socket 操作を伴うため、**Linux (Ubuntu 22.04/24.04 推奨)** 環境が必須です。

### 1. 必要なツール

```bash
sudo apt update && sudo apt install -y golang-go gcc tcpdump wireshark iproute2
```

### 2. プロジェクト構成

```bash
mkdir -p tcpip_stack/pkg/{rawsock,ethernet,ipv4,icmp}
mkdir -p tcpip_stack/cmd/ping
cd tcpip_stack
go mod init tcpip_stack
```

### 3. 安全な実験場の作成 (Network Namespace)

Raw Socketを用いた開発では、誤ったパケットを送出するとホストOSのルーティングテーブルを混乱させるリスクがあります。Network Namespace で隔離環境を作ります。

```bash
# 1. 隔離された「砂場」を作成
sudo ip netns add workshop

# 2. 仮想LANケーブル (veth) を作成
sudo ip link add veth-host type veth peer name veth-ns

# 3. 片方の端点を隔離空間に移動
sudo ip link set veth-ns netns workshop

# 4. 両端のインターフェースを起動
sudo ip link set veth-host up
sudo ip netns exec workshop ip link set veth-ns up

# 5. IPアドレスを割り当て
sudo ip addr add 192.168.100.1/24 dev veth-host
sudo ip netns exec workshop ip addr add 192.168.100.2/24 dev veth-ns
```

---

## 🚀 実装ステップ

### STEP 1: Raw Socket (L1) - カーネルをバイパスする

#### 目標

通常のアプリケーションは `TCP/UDP Socket` を使いますが、これでは OS が L2/L3 ヘッダを自動生成してしまいます。自作スタックでは **AF_PACKET** を使い、NIC から直接「生データ」を読み書きします。

#### 実装: C ヘッダファイル (`pkg/rawsock/rawsock.h`)

```c
#ifndef RAWSOCK_H
#define RAWSOCK_H

#ifdef __cplusplus
extern "C" {
#endif

// Raw Socket を開く
// 戻り値: ファイルディスクリプタ、エラー時は -1
int rawsock_open(const char *iface);

// パケットを受信
// 戻り値: 受信バイト数、エラー時は -1
int rawsock_recv(int fd, void *buf, int len);

// パケットを送信
// 戻り値: 送信バイト数、エラー時は -1
int rawsock_send(int fd, const void *buf, int len, const char *iface);

// Raw Socket を閉じる
void rawsock_close(int fd);

// インターフェースの MAC アドレスを取得
int rawsock_get_mac(const char *iface, unsigned char *mac);

// インターフェースの MTU を取得
int rawsock_get_mtu(const char *iface);

// プロミスキャスモードを設定
int rawsock_set_promisc(int fd, const char *iface, int enable);

#ifdef __cplusplus
}
#endif

#endif // RAWSOCK_H
```

#### 実装: C 実装ファイル (`pkg/rawsock/rawsock.c`)

```c
#include "rawsock.h"
#include <sys/socket.h>
#include <sys/ioctl.h>
#include <linux/if_ether.h>
#include <linux/if_packet.h>
#include <net/if.h>
#include <unistd.h>
#include <string.h>
#include <arpa/inet.h>
#include <stdio.h>

int rawsock_open(const char *iface) {
    // AF_PACKET: データリンク層（L2）に直接アクセス
    // SOCK_RAW: ヘッダを含む全パケットを取得
    // ETH_P_ALL: 全てのプロトコルタイプを受信
    int fd = socket(AF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
    if (fd < 0) {
        perror("socket");
        return -1;
    }

    // 指定インターフェースにバインド
    struct sockaddr_ll addr;
    memset(&addr, 0, sizeof(addr));
    addr.sll_family = AF_PACKET;
    addr.sll_protocol = htons(ETH_P_ALL);
    addr.sll_ifindex = if_nametoindex(iface);

    if (addr.sll_ifindex == 0) {
        perror("if_nametoindex");
        close(fd);
        return -1;
    }

    if (bind(fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("bind");
        close(fd);
        return -1;
    }

    return fd;
}

int rawsock_recv(int fd, void *buf, int len) {
    return recv(fd, buf, len, 0);
}

int rawsock_send(int fd, const void *buf, int len, const char *iface) {
    struct sockaddr_ll addr;
    memset(&addr, 0, sizeof(addr));
    addr.sll_family = AF_PACKET;
    addr.sll_ifindex = if_nametoindex(iface);

    if (addr.sll_ifindex == 0) {
        return -1;
    }

    return sendto(fd, buf, len, 0, (struct sockaddr*)&addr, sizeof(addr));
}

void rawsock_close(int fd) {
    if (fd >= 0) {
        close(fd);
    }
}

int rawsock_get_mac(const char *iface, unsigned char *mac) {
    int fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) {
        return -1;
    }

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, iface, IFNAMSIZ - 1);

    // ioctl でハードウェアアドレスを取得
    if (ioctl(fd, SIOCGIFHWADDR, &ifr) < 0) {
        perror("ioctl SIOCGIFHWADDR");
        close(fd);
        return -1;
    }

    memcpy(mac, ifr.ifr_hwaddr.sa_data, 6);
    close(fd);
    return 0;
}

int rawsock_get_mtu(const char *iface) {
    int fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) {
        return -1;
    }

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, iface, IFNAMSIZ - 1);

    if (ioctl(fd, SIOCGIFMTU, &ifr) < 0) {
        close(fd);
        return -1;
    }

    int mtu = ifr.ifr_mtu;
    close(fd);
    return mtu;
}

int rawsock_set_promisc(int fd, const char *iface, int enable) {
    struct packet_mreq mr;
    memset(&mr, 0, sizeof(mr));
    mr.mr_ifindex = if_nametoindex(iface);
    mr.mr_type = PACKET_MR_PROMISC;

    if (setsockopt(fd, SOL_PACKET, PACKET_ADD_MEMBERSHIP, &mr, sizeof(mr)) < 0) {
        perror("setsockopt PACKET_ADD_MEMBERSHIP");
        return -1;
    }

    return 0;
}
```

#### 実装: Go バインディング (`pkg/rawsock/rawsock.go`)

```go
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

// Socket は Raw Socket を表す
type Socket struct {
	fd    C.int
	iface string
}

// Open は指定インターフェースで Raw Socket を開く
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

// Recv はパケットを受信してバッファに書き込む
// 重要: unsafe.Pointer を使って Go スライスに直接書き込み、コピーを回避
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

// Send はパケットを送信する
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

// Close は Raw Socket を閉じる
func (s *Socket) Close() error {
	C.rawsock_close(s.fd)
	s.fd = -1
	return nil
}

// GetMAC はインターフェースの MAC アドレスを取得
func GetMAC(iface string) (net.HardwareAddr, error) {
	cIface := C.CString(iface)
	defer C.free(unsafe.Pointer(cIface))

	var mac [6]C.uchar
	ret := C.rawsock_get_mac(cIface, &mac[0])
	if ret < 0 {
		return nil, fmt.Errorf("failed to get MAC address")
	}

	return net.HardwareAddr{
		byte(mac[0]), byte(mac[1]), byte(mac[2]),
		byte(mac[3]), byte(mac[4]), byte(mac[5]),
	}, nil
}
```

#### 学習ポイント

##### AF_PACKET とは？

AF_PACKET は Linux で **データリンク層（L2）に直接アクセスする** ためのソケットファミリです。通常のソケット（AF_INET）ではカーネルが自動的に TCP/IP ヘッダを処理しますが、AF_PACKET を使うと **Ethernet フレーム全体を生のバイト列として** 送受信できます。

```
Regular Socket (AF_INET)
+-----------+     +------------------------+
|   App     | <-- | Kernel handles TCP/IP  |
+-----------+     +------------------------+

Raw Socket (AF_PACKET)
+-----------+     +------------------------------------+
|   App     | <-- | Raw Ethernet frame (bytes)         |
+-----------+     | Build headers from MAC layer up    |
                  +------------------------------------+
```

##### socket() システムコールの引数

```c
int fd = socket(AF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
//              │          │         │
//              │          │         └─ プロトコル: 全てのパケットを受信
//              │          └─ タイプ: ヘッダを含む生データ
//              └─ ファミリ: データリンク層アクセス
```

| 引数 | 値 | 意味 |
|------|-----|------|
| ドメイン | `AF_PACKET` | パケットレベルのソケット（L2） |
| タイプ | `SOCK_RAW` | ヘッダを含む生パケットを取得 |
| プロトコル | `ETH_P_ALL` (0x0003) | 全プロトコルタイプを受信 |

##### 他のプロトコル値の例

| プロトコル | 値 | 説明 |
|-----------|-------|------|
| `ETH_P_ALL` | 0x0003 | 全パケット受信 |
| `ETH_P_IP` | 0x0800 | IPv4 のみ受信 |
| `ETH_P_IPV6` | 0x86DD | IPv6 のみ受信 |
| `ETH_P_ARP` | 0x0806 | ARP のみ受信 |

##### なぜ AF_PACKET が必要か？

自作スタックを作る場合、以下の理由で AF_PACKET が必須です：

1. **ヘッダの自前生成**: Ethernet、IP、ICMP ヘッダを自分で構築したい
2. **カーネルのバイパス**: カーネルのネットワークスタックを経由せず直接 NIC と通信
3. **カスタムプロトコル**: 標準的でないプロトコルを実装したい
4. **パケット解析**: Wireshark のようなネットワークモニターを作りたい

##### 注意点

- **root 権限が必要**: セキュリティ上の理由で `CAP_NET_RAW` ケーパビリティが必要
- **Linux 専用**: macOS は BPF (Berkeley Packet Filter)、Windows は WinPcap/Npcap を使用
- **誤送信のリスク**: 間違ったパケットを送るとネットワーク全体に影響する可能性

##### CGO の役割

Go から C のシステムコールを直接呼ぶために CGO を使用します：

```go
/*
#cgo LDFLAGS: -lpcap
#include "rawsock.h"
*/
import "C"

// unsafe.Pointer で Go のメモリを C に直接渡す（コピー回避）
n := C.rawsock_recv(s.fd, unsafe.Pointer(&buf[0]), C.int(len(buf)))
```

- **ゼロコピー**: `unsafe.Pointer` で Go スライスのアドレスを直接 C に渡し、メモリコピーを回避
- **パフォーマンス**: パケット処理では μ秒単位の速度が重要

##### プロミスキャスモード

通常、NIC は自分の MAC アドレス宛てまたはブロードキャストのパケットのみを渡します。プロミスキャスモードを有効にすると、**全てのパケット** を受信できます。

```c
setsockopt(fd, SOL_PACKET, PACKET_ADD_MEMBERSHIP, &mr, sizeof(mr));
```

---

### STEP 2: Ethernet Frame (L2) - MAC アドレスベースの通信

#### 目標

Ethernet は物理的に隣接する機器間の通信を制御します。MAC アドレス（48bit）を使って、同一セグメント内の通信相手を特定します。

#### Ethernet II フレーム構造

```
┌───────────────────┬───────────────────┬──────────────┬─────────────┐
│  Dst MAC (6bytes) │  Src MAC (6bytes) │ Type (2B)    │ Payload     │
├───────────────────┴───────────────────┴──────────────┴─────────────┤
│                          Minimum 60 bytes                          │
└────────────────────────────────────────────────────────────────────┘
```

#### 実装: Ethernet Frame (`pkg/ethernet/frame.go`)

```go
package ethernet

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// Ethernet II ヘッダサイズ
	HeaderSize = 14 // 6 (DstMAC) + 6 (SrcMAC) + 2 (EtherType)

	// 最小フレームサイズ（パディング含む）
	MinFrameSize = 60

	// 最大フレームサイズ
	MaxFrameSize = 1514

	// EtherType 値
	TypeIPv4 = 0x0800
	TypeARP  = 0x0806
	TypeIPv6 = 0x86DD
)

// Frame は Ethernet II フレームを表す
type Frame struct {
	DstMAC    net.HardwareAddr
	SrcMAC    net.HardwareAddr
	EtherType uint16
	Payload   []byte
}

// Parse はバイト列から Ethernet フレームを解析
func Parse(data []byte) (*Frame, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("packet too short: %d < %d", len(data), HeaderSize)
	}

	return &Frame{
		DstMAC:    net.HardwareAddr(data[0:6]),  // 宛先 MAC
		SrcMAC:    net.HardwareAddr(data[6:12]), // 送信元 MAC
		EtherType: binary.BigEndian.Uint16(data[12:14]), // プロトコルタイプ
		Payload:   data[14:], // ペイロード（上位レイヤーのデータ）
	}, nil
}

// Marshal はフレームをバイト列に変換
func (f *Frame) Marshal() []byte {
	payloadLen := len(f.Payload)
	totalLen := HeaderSize + payloadLen

	buf := make([]byte, totalLen)

	// MAC アドレスをコピー
	copy(buf[0:6], f.DstMAC)
	copy(buf[6:12], f.SrcMAC)

	// EtherType をセット（ネットワークバイトオーダ = Big Endian）
	binary.BigEndian.PutUint16(buf[12:14], f.EtherType)

	// ペイロードをコピー
	if payloadLen > 0 {
		copy(buf[14:], f.Payload)
	}

	// 最小フレームサイズに満たない場合はパディング
	// Ethernet の物理層は 60 bytes 未満のフレームを正しく処理できない
	if totalLen < MinFrameSize {
		padded := make([]byte, MinFrameSize)
		copy(padded, buf)
		return padded
	}

	return buf
}

// BroadcastMAC はブロードキャスト MAC アドレスを返す
func BroadcastMAC() net.HardwareAddr {
	return net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
}

// IsBroadcast はブロードキャストアドレスかどうかを判定
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

// IsMulticast はマルチキャストアドレスかどうかを判定
// マルチキャスト MAC は最下位ビットが 1
func IsMulticast(mac net.HardwareAddr) bool {
	if len(mac) != 6 {
		return false
	}
	return mac[0]&0x01 == 0x01
}
```

#### 学習ポイント

- **ネットワークバイトオーダ**: ネットワーク通信では Big Endian を使用
- **パディング**: Ethernet フレームは最低 60 bytes 必要。短い場合は 0 埋め
- **EtherType**: ペイロードのプロトコルを識別（0x0800 = IPv4）

---

### STEP 3: IPv4 Packet (L3) - IP アドレスベースのルーティング

#### 目標

IPv4 は異なるネットワークセグメント間での通信を可能にします。32bit の IP アドレスで通信先を特定し、チェックサムでヘッダの整合性を検証します。

#### IPv4 ヘッダ構造

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Version|  IHL  |    TOS       |          Total Length         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Identification        |Flags|     Fragment Offset     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  TTL  | Protocol|        Header Checksum                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Source Address                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Destination Address                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

#### 実装: IPv4 Packet (`pkg/ipv4/packet.go`)

```go
package ipv4

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// 最小ヘッダサイズ（オプションなし）
	HeaderSize = 20

	// 最大パケットサイズ
	MaxPacketSize = 65535

	// プロトコル番号
	ProtocolICMP = 1
	ProtocolTCP  = 6
	ProtocolUDP  = 17

	// フラグ
	FlagDF = 0x01 // Don't Fragment（フラグメント禁止）
	FlagMF = 0x02 // More Fragments（後続フラグメントあり）
)

// Packet は IPv4 パケットを表す
type Packet struct {
	Version        uint8  // 常に 4
	IHL            uint8  // Internet Header Length (4 bytes 単位)
	TOS            uint8  // Type of Service
	TotalLength    uint16 // ヘッダ + ペイロード
	ID             uint16 // 識別子（フラグメント再構築用）
	Flags          uint8
	FragmentOffset uint16 // フラグメントオフセット（8 bytes 単位）
	TTL            uint8  // Time to Live（ホップ数制限）
	Protocol       uint8  // 上位プロトコル
	Checksum       uint16 // ヘッダチェックサム
	SrcIP          net.IP
	DstIP          net.IP
	Options        []byte // IP オプション（通常は使用しない）
	Payload        []byte
}

// Parse はバイト列から IPv4 パケットを解析
func Parse(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("packet too short: %d < %d", len(data), HeaderSize)
	}

	// バージョンチェック
	version := data[0] >> 4
	if version != 4 {
		return nil, fmt.Errorf("not IPv4: version=%d", version)
	}

	// IHL (Internet Header Length) は 4 bytes 単位
	ihl := (data[0] & 0x0F) * 4
	if len(data) < int(ihl) {
		return nil, fmt.Errorf("packet too short for IHL: %d < %d", len(data), ihl)
	}

	// Flags と Fragment Offset は 16bit にパックされている
	//   Flags: 上位 3bit
	//   Fragment Offset: 下位 13bit
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
		Payload:        data[ihl:], // ペイロードは IHL 以降
	}

	if ihl > HeaderSize {
		pkt.Options = data[HeaderSize:ihl]
	}

	return pkt, nil
}

// Marshal はパケットをバイト列に変換
func (p *Packet) Marshal() []byte {
	ihl := HeaderSize
	if len(p.Options) > 0 {
		ihl = HeaderSize + len(p.Options)
	}

	payloadLen := len(p.Payload)
	totalLen := ihl + payloadLen

	buf := make([]byte, ihl)

	// Version (4bit) + IHL (4bit)
	buf[0] = (4 << 4) | (uint8(ihl/4) & 0x0F)
	buf[1] = p.TOS
	binary.BigEndian.PutUint16(buf[2:4], uint16(totalLen))
	binary.BigEndian.PutUint16(buf[4:6], p.ID)

	// Flags (3bit) + Fragment Offset (13bit)
	flagsFragment := (uint16(p.Flags) << 13) | (p.FragmentOffset & 0x1FFF)
	binary.BigEndian.PutUint16(buf[6:8], flagsFragment)

	buf[8] = p.TTL
	buf[9] = p.Protocol
	// Checksum は後で計算

	// IP アドレス（4bytes に変換）
	copy(buf[12:16], p.SrcIP.To4())
	copy(buf[16:20], p.DstIP.To4())

	// オプション
	if len(p.Options) > 0 {
		copy(buf[20:], p.Options)
	}

	// チェックサム計算
	checksum := Checksum(buf[:ihl])
	binary.BigEndian.PutUint16(buf[10:12], checksum)

	// ペイロード追加
	result := append(buf, p.Payload...)

	return result
}

// Checksum は IPv4 ヘッダのチェックサムを計算
// RFC 1071: 16bit の 1 の補数和の 1 の補数
func Checksum(data []byte) uint16 {
	sum := uint32(0)

	// 16bit 単位で加算
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// 奇数長の場合の処理
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// 32bit → 16bit に畳み込み（キャリーを加算）
	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// 1 の補数を返す
	return ^uint16(sum)
}

// IsFragmented はパケットがフラグメント化されているかを判定
func (p *Packet) IsFragmented() bool {
	return p.FragmentOffset != 0 || p.Flags&FlagMF != 0
}

// PseudoHeader は TCP/UDP チェックサム計算用の疑似ヘッダを生成
func (p *Packet) PseudoHeader() []byte {
	buf := make([]byte, 12)
	copy(buf[0:4], p.SrcIP.To4())
	copy(buf[4:8], p.DstIP.To4())
	buf[8] = 0
	buf[9] = p.Protocol
	binary.BigEndian.PutUint16(buf[10:12], uint16(len(p.Payload)))
	return buf
}
```

#### 学習ポイント

- **IHL (Internet Header Length)**: ヘッダ長を 4 bytes 単位で表現（通常は 5 = 20 bytes）
- **TTL**: ループ防止用。ルータを通過するたびに減算され、0 で破棄
- **チェックサム**: ヘッダの破損を検出。送信時に計算、受信時に検証

---

### STEP 4: ICMP Message (L4) - Ping の実装

#### 目標

ICMP (Internet Control Message Protocol) はネットワーク診断に使用されます。最も一般的な用途は「Ping」（Echo Request/Reply）です。

#### ICMP Echo メッセージ構造

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |     Code      |          Checksum             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Identifier          |        Sequence Number        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                             Data                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

#### 実装: ICMP Message (`pkg/icmp/message.go`)

```go
package icmp

import (
	"encoding/binary"
	"fmt"

	"tcpip_stack/pkg/ipv4"
)

const (
	// ICMP タイプ
	EchoReply              = 0
	DestinationUnreachable = 3
	EchoRequest            = 8
	TimeExceeded           = 11
)

// Message は ICMP メッセージを表す
type Message struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16 // Echo Request/Reply 用
	Seq      uint16 // Echo Request/Reply 用
	Data     []byte
}

// Parse はバイト列から ICMP メッセージを解析
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

// Marshal はメッセージをバイト列に変換
func (m *Message) Marshal() []byte {
	buf := make([]byte, 8+len(m.Data))

	buf[0] = m.Type
	buf[1] = m.Code
	// Checksum は後で計算
	binary.BigEndian.PutUint16(buf[4:6], m.ID)
	binary.BigEndian.PutUint16(buf[6:8], m.Seq)

	if len(m.Data) > 0 {
		copy(buf[8:], m.Data)
	}

	// チェックサム計算（IPv4 と同じアルゴリズム）
	checksum := ipv4.Checksum(buf)
	binary.BigEndian.PutUint16(buf[2:4], checksum)

	return buf
}

// NewEchoRequest は ICMP Echo Request を作成
func NewEchoRequest(id, seq uint16, data []byte) *Message {
	return &Message{
		Type: EchoRequest,
		Code: 0,
		ID:   id,
		Seq:  seq,
		Data: data,
	}
}

// NewEchoReply は ICMP Echo Reply を作成
func NewEchoReply(id, seq uint16, data []byte) *Message {
	return &Message{
		Type: EchoReply,
		Code: 0,
		ID:   id,
		Seq:  seq,
		Data: data,
	}
}

// IsEchoRequest は Echo Request かどうかを判定
func (m *Message) IsEchoRequest() bool {
	return m.Type == EchoRequest
}

// IsEchoReply は Echo Reply かどうかを判定
func (m *Message) IsEchoReply() bool {
	return m.Type == EchoReply
}
```

#### 学習ポイント

- **Type 8/0**: Echo Request (8) に対して Echo Reply (0) を返す
- **ID/Seq**: 複数の Ping を区別するための識別子
- **Data**: 往復時間（RTT）計測用のタイムスタンプなどを格納

---

### STEP 5: Stack 統合 (`main.go`) - 全てを繋げる

#### 目標

これまでのコンポーネントを統合し、実際に Ping に応答するサーバーを構築します。

#### 実装: メインプログラム (`main.go`)

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"tcpip_stack/pkg/ethernet"
	"tcpip_stack/pkg/icmp"
	"tcpip_stack/pkg/ipv4"
	"tcpip_stack/pkg/rawsock"
)

// Stack は TCP/IP スタックを表す
type Stack struct {
	sock   *rawsock.Socket
	iface  string
	srcMAC net.HardwareAddr
	srcIP  net.IP
}

// NewStack は新しいスタックを作成
func NewStack(iface string) (*Stack, error) {
	// Raw Socket を開く
	sock, err := rawsock.Open(iface)
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

	// インターフェース情報を取得
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get interface: %w", err)
	}

	// IP アドレスを取得
	addrs, err := ifaceObj.Addrs()
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get addresses: %w", err)
	}

	var srcIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			if !ipnet.IP.IsLoopback() {
				srcIP = ipnet.IP
				break
			}
		}
	}

	if srcIP == nil {
		sock.Close()
		return nil, fmt.Errorf("no IPv4 address found for interface %s", iface)
	}

	return &Stack{
		sock:   sock,
		iface:  iface,
		srcMAC: ifaceObj.HardwareAddr,
		srcIP:  srcIP,
	}, nil
}

// Run はパケット処理ループを開始
func (s *Stack) Run(ctx context.Context) error {
	buf := make([]byte, 65536)
	log.Printf("Listening on %s (%s, MAC: %s)", s.iface, s.srcIP, s.srcMAC)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// パケット受信
			n, err := s.sock.Recv(buf)
			if err != nil {
				continue
			}

			// バイト列をコピー（次の受信で上書きされるため）
			packetData := make([]byte, n)
			copy(packetData, buf[:n])

			if err := s.handlePacket(packetData); err != nil {
				log.Printf("handle packet: %v", err)
			}
		}
	}
}

// handlePacket は受信パケットを処理
func (s *Stack) handlePacket(data []byte) error {
	// L2: Ethernet フレーム解析
	frame, err := ethernet.Parse(data)
	if err != nil {
		return err
	}

	// 自分宛でないパケットは無視
	if !s.shouldProcessFrame(frame) {
		return nil
	}

	// IPv4 以外は無視
	if frame.EtherType != ethernet.TypeIPv4 {
		return nil
	}

	// L3: IPv4 パケット解析
	pkt, err := ipv4.Parse(frame.Payload)
	if err != nil {
		return err
	}

	// ICMP 以外は無視
	if pkt.Protocol != ipv4.ProtocolICMP {
		return nil
	}

	// L4: ICMP メッセージ解析
	msg, err := icmp.Parse(pkt.Payload)
	if err != nil {
		return err
	}

	// Ping (Echo Request) に応答
	if msg.IsEchoRequest() && pkt.DstIP.Equal(s.srcIP) {
		log.Printf("Ping from %s: ID=%d Seq=%d", pkt.SrcIP, msg.ID, msg.Seq)
		return s.sendEchoReply(pkt, msg, frame.SrcMAC)
	}

	return nil
}

// shouldProcessFrame はフレームを処理すべきか判定
func (s *Stack) shouldProcessFrame(frame *ethernet.Frame) bool {
	if frame.DstMAC.String() == s.srcMAC.String() {
		return true // ユニキャスト
	}
	if ethernet.IsBroadcast(frame.DstMAC) {
		return true // ブロードキャスト
	}
	if ethernet.IsMulticast(frame.DstMAC) {
		return true // マルチキャスト
	}
	return false
}

// sendEchoReply は ICMP Echo Reply を送信
func (s *Stack) sendEchoReply(reqPkt *ipv4.Packet, reqMsg *icmp.Message, dstMAC net.HardwareAddr) error {
	// L4: ICMP Echo Reply を作成
	reply := icmp.NewEchoReply(reqMsg.ID, reqMsg.Seq, reqMsg.Data)
	icmpData := reply.Marshal()

	// L3: IPv4 パケットを作成
	ipPkt := &ipv4.Packet{
		Version:  4,
		IHL:      20,
		TOS:      0,
		TTL:      64,
		Protocol: ipv4.ProtocolICMP,
		SrcIP:    s.srcIP,
		DstIP:    reqPkt.SrcIP,
		Payload:  icmpData,
	}
	ipData := ipPkt.Marshal()

	// L2: Ethernet フレームを作成
	frame := &ethernet.Frame{
		DstMAC:    dstMAC,
		SrcMAC:    s.srcMAC,
		EtherType: ethernet.TypeIPv4,
		Payload:   ipData,
	}
	frameData := frame.Marshal()

	// 送信
	_, err := s.sock.Send(frameData)
	return err
}

// Close はスタックを閉じる
func (s *Stack) Close() error {
	return s.sock.Close()
}

func main() {
	iface := flag.String("iface", "", "Network interface to bind (required)")
	flag.Parse()

	if *iface == "" {
		fmt.Println("TCP/IP Protocol Stack Workshop")
		fmt.Println("Usage: sudo go run main.go -iface <interface>")
		os.Exit(1)
	}

	stack, err := NewStack(*iface)
	if err != nil {
		log.Fatalf("NewStack: %v", err)
	}
	defer stack.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("TCP/IP stack running on %s (%s)", *iface, stack.srcIP)
	log.Printf("Try: ping %s", stack.srcIP)

	if err := stack.Run(ctx); err != nil {
		log.Fatalf("Run: %v", err)
	}

	log.Println("TCP/IP stack shut down gracefully")
}
```

---

## 🧪 動作確認

### 1. ビルド

```bash
go mod tidy
go build -o tcpip_stack main.go
```

### 2. スタックの起動

```bash
# 作成した veth のホスト側で起動
sudo ./tcpip_stack -iface veth-host
```

### 3. Ping テスト

別のターミナルから:

```bash
# Namespace 内から Ping
sudo ip netns exec workshop ping 192.168.100.1
```

### 4. パケット観察

```bash
sudo tcpdump -i veth-host -nn -vv icmp
```

---

## 📊 パケットフロー図

```
=================== Receive Path (Rx) ===================

NIC --> Raw Socket --> Ethernet.Parse --> IPv4.Parse --> ICMP.Parse
                           |                  |               |
                      [MAC Filter]      [IP Filter]    [Type Check]
                           |                  |               |
                        Pass?              Pass?           Type=8?
                           |                  |               |
                           v                  v               v
                      Drop/Continue     Drop/Continue   Drop/Reply

==================== Send Path (Tx) ====================

NIC <-- Raw Socket <-- Ethernet.Marshal <-- IPv4.Marshal <-- ICMP.Reply
```

**凡例:**

- `-->` : データの流れ（パケット/バイト列）
- `|` : 処理の分岐点
- `v` : 次の処理へ
- `[]` : フィルタリング処理

### パケット処理の詳細な流れ

#### 1. 受信パス（左から右へ）

Ping 要求が届いた時の処理フロー:

```
ステップ 1: NIC → Raw Socket
───────────────────────────
・NIC が電気信号をビット列に変換
・Raw Socket がバイト列として受信
・データ: [Ethernet][IPv4][ICMP][Data] の生バイト列

ステップ 2: Ethernet.Parse
───────────────────────────
・バイト列から Ethernet ヘッダを解析
・抽出: DstMAC, SrcMAC, EtherType
・[MAC Filter]: DstMAC が自分宛か確認
  - 自分のMAC または ブロードキャスト → 継続
  - それ以外 → 破棄（処理終了）

ステップ 3: IPv4.Parse
───────────────────────────
・Ethernet ペイロードから IPv4 ヘッダを解析
・抽出: SrcIP, DstIP, Protocol, TTL 等
・[IP Filter]: Protocol が ICMP か確認
  - Protocol = 1 (ICMP) → 継続
  - それ以外 → 破棄（TCP/UDP は今回は無視）

ステップ 4: ICMP.Parse
───────────────────────────
・IPv4 ペイロードから ICMP メッセージを解析
・抽出: Type, Code, ID, Seq, Data
・[Type Check]: Type が Echo Request (8) か確認
  - Type = 8 → Echo Reply を生成して送信パスへ
  - それ以外 → 破棄
```

#### 2. 送信パス（右から左へ）

Echo Reply を返す時の処理フロー:

```
ステップ 1: ICMP.Reply 生成
───────────────────────────
・受信した Echo Request から Reply を作成
・Type: 8 → 0 に変更
・ID, Seq, Data: そのままコピー
・Checksum: 再計算

ステップ 2: IPv4.Marshal
───────────────────────────
・IPv4 パケットをバイト列に変換
・SrcIP/DstIP を反転（要求元 → 自分）
・TTL: 64 を設定
・Checksum: 計算して設定

ステップ 3: Ethernet.Marshal
───────────────────────────
・Ethernet フレームをバイト列に変換
・DstMAC/SrcMAC を反転（要求元 → 自分）
・EtherType: 0x0800 (IPv4)

ステップ 4: Raw Socket → NIC
───────────────────────────
・完成したバイト列を Raw Socket に渡す
・NIC が電気信号として送信
```

### 具体例：Ping 要求〜応答のシーケンス

```
T1: Namespace 側で ping 192.168.100.1 実行
    |
    v
T2: Packet arrives (Echo Request)
    +---------------------------------------------------+
    | Ethernet: Dst=FF:FF:FF:FF:FF:FF                   |
    | IPv4:     Src=192.168.100.2, Dst=192.168.100.1    |
    | ICMP:     Type=8, ID=1234, Seq=1                  |
    +---------------------------------------------------+
    |
    v  veth-host に到着
    |
T3: Custom stack receives
    |  -> MAC Filter: broadcast -> Pass
    |  -> IP Filter:  DstIP is ours -> Pass
    |  -> Type Check: Type=8 -> Generate Reply
    |
    v
T4: Packet sent (Echo Reply)
    +---------------------------------------------------+
    | Ethernet: Dst=xx:xx:xx:xx:xx:xx                   |
    | IPv4:     Src=192.168.100.1, Dst=192.168.100.2    |
    | ICMP:     Type=0, ID=1234, Seq=1                  |
    +---------------------------------------------------+
    |
    v  Namespace 側に送信
    |
T5: ping receives Reply
    "64 bytes from 192.168.100.1: icmp_seq=1 ttl=64"
```

---

## 🧹 後片付け

```bash
sudo ip link delete veth-host
sudo ip netns delete workshop
```

---

## 💡 深掘りポイント

1. **Zero-Copy**: CGO で `unsafe.Pointer` を使いコピーを回避
2. **チェックサム**: RFC 1071 の 1 の補数和アルゴリズム
3. **エンディアン**: ネットワークバイトオーダ (Big Endian) の重要性
4. **拡張性**: TCP/UDP を追加する場合の設計

## 📚 参考文献

- [RFC 791 (IPv4)](https://datatracker.ietf.org/doc/html/rfc791)
- [RFC 792 (ICMP)](https://datatracker.ietf.org/doc/html/rfc792)
- [RFC 1071 (Checksum)](https://datatracker.ietf.org/doc/html/rfc1071)
