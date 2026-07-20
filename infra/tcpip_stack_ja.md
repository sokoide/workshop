# TCP/IP プロトコルスタック実習：スクラッチから構築する「通信の正体」

この実習では、Go と C を組み合わせて **TCP/IP プロトコルスタックをゼロから構築** します。OS カーネルが隠蔽している「パケットの生成・分解」を自らの手で実装することで、ネットワークの動作原理を抽象概念ではなく、具体的なデータ構造として理解することを目標とします。

> **💡 用語集**: この実習で登場する専門用語は [用語集](glossary_ja.md) を参照してください。

---

## 🏗 アーキテクチャ：ロシアのマトリョーシカ（カプセル化）

ネットワーク通信の本質は「カプセル化」です。上位レイヤーのデータが下位レイヤーの「ペイロード」として包まれていきます。

```text
+----------------------------------------------------+
| Ethernet Frame (L2)                                |
| +------------------------------------------------+ |
|                                                    | IPv4 Packet (L3)                               |                                            |
|                                                    | +--------------------------------------------+ |                                            |
|                                                    |                                                | ICMP Message (L4)                          |                |  |
|                                                    |                                                | +----------------------------------------+ |                |  |
|                                                    |                                                |                                            | Payload (Data) |  |  |  |
|                                                    |                                                | +----------------------------------------+ |                |  |
|                                                    | +--------------------------------------------+ |                                            |
| +------------------------------------------------+ |
+----------------------------------------------------+
```

### 今回実装するコンポーネント

| レイヤー | コンポーネント | ファイル                | 実装内容                                         |
| :------- | :------------- | :---------------------- | :----------------------------------------------- |
| L2       | Packet Socket  | `pkg/rawsock/`          | Ethernet フレームを送受信する Linux の AF_PACKET |
| L2       | Ethernet Frame | `pkg/ethernet/frame.go` | MAC アドレスベースのフレーム処理                 |
| L3       | IPv4 Packet    | `pkg/ipv4/packet.go`    | IP アドレスベースのパケット処理                  |
| L4       | ICMP Message   | `pkg/icmp/message.go`   | Ping (Echo Request/Reply) の実装                 |
| L4       | UDP Message    | `pkg/udp/message.go`    | コネクションレスデータグラム                     |
| App      | Stack          | `main.go`               | 全体を統合するサーバー                           |

---

## 🛠 開発環境の準備

本実習は、低レベルな Raw Socket 操作を伴うため、**Linux (Ubuntu 22.04/24.04 推奨)** 環境が必須です。

### 1. 必要なツール

```bash
sudo apt update && sudo apt install -y golang-go gcc tcpdump wireshark iproute2
```

### 2. プロジェクト構成

```bash
mkdir -p tcpip_stack/pkg/{rawsock,ethernet,ipv4,icmp,udp}
mkdir -p tcpip_stack/cmd/ping
cd tcpip_stack
go mod init github.com/sokoide/workshop/infra/assets/tcpip_stack
```

### 3. 安全な実験場の作成 (Network Namespace)

この実習では Linux の IP スタックが応答を横取りしないよう、`veth-host` には IP アドレスを設定しません。自作スタックだけが `192.168.100.1` を名乗り、Namespace 側に静的 ARP エントリを設定します。これは ARP の実装を後続課題として切り出すための意図的な省略です。

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

# 5. クライアント側だけに IP アドレスを割り当てる
sudo ip netns exec workshop ip addr add 192.168.100.2/24 dev veth-ns

# 6. 自作スタック側の MAC を ARP キャッシュに固定する
STACK_MAC=$(cat /sys/class/net/veth-host/address)
sudo ip netns exec workshop ip neigh replace 192.168.100.1 lladdr "$STACK_MAC" nud permanent dev veth-ns
```

### ✅ チェックポイント

- [ ] `ip netns list` で `workshop` が表示される
- [ ] `ip addr show veth-host` に IPv4 アドレスがない
- [ ] `sudo ip netns exec workshop ip addr show veth-ns` で `192.168.100.2` が割り当てられている
- [ ] `sudo ip netns exec workshop ip neigh show` に `192.168.100.1` の permanent エントリがある

---

## 🚀 実装ステップ

### STEP 0: 通信を観察する — カーネルが隠しているもの

#### 目標

実装に入る前に、普段見えない「通信の中身」を観察します。`curl` 1 行でアプリが得る情報と、`tcpdump` で見える生のパケット列を比較することで、この実習の意義（＝なぜパケットを自作するのか）を実感します。

#### 1. アプリから見た通信（curl -v）

ローカルに HTTP サーバを立ててリクエストを観察します。

```bash
# 別ターミナル: 最小のHTTPサーバを起動
python3 -m http.server 8000

# メインのターミナル: 詳細ログ付きでリクエスト
curl -v http://localhost:8000/
```

`curl -v` が表示するのは、HTTP という「人間が読めるプロトコル」のやり取りです。

```text
*   Trying 127.0.0.1:8000...
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET / HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/8.x
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: SimpleHTTP/0.6 Python/3.x
< Content-Length: 1234
<
(以下、HTML本体)
```

ここには **TCP接続（3ウェイハンドシェイク）も、IPアドレスのカプセル化も、Ethernetフレームも見えません。** カーネルが全て隠してくれているからです。

#### 2. カーネルが実際に送っているもの（tcpdump）

同じリクエストをパケットレベルで観察します。

```bash
# 別ターミナル: ループバックを観察開始
sudo tcpdump -i lo -nn -vv 'tcp port 8000'

# もう一度リクエスト
curl -s http://localhost:8000/ > /dev/null
```

```text
13:00:01.123456 IP (tos 0x0, ttl 64, id 0, offset 0, flags [DF], proto TCP (6), length 60)
    127.0.0.1.54321 > 127.0.0.1.8000: Flags [S], cksum 0xfe34, seq 123456789, win 65535
13:00:01.123489 IP (... length 60)
    127.0.0.1.8000 > 127.0.0.1.54321: Flags [S.], cksum 0xfe34, seq 987654321, ack 123456790, win 65535
13:00:01.123501 IP (... length 52)
    127.0.0.1.54321 > 127.0.0.1.8000: Flags [.], cksum 0xfe34, ack 1, win 65535
13:00:01.123590 IP (... length 100)
    127.0.0.1.54321 > 127.0.0.1.8000: Flags [P.], seq 1:49, ack 1, win 65535, length 48: HTTP, length: 48
    GET / HTTP/1.1
```

これが「通信の正体」です。1 回の `curl` の裏で:

- `[S]` `[S.]` `[.]` = **3ウェイハンドシェイク** (SYN, SYN-ACK, ACK)
- `[P.]` = データ搭載パケット (PSH-ACK)
- さらに ACK、FIN、FIN-ACK… と続く

TCP は「信頼性」を保証するために、このような複雑な状態機械を動かしています。

#### 学習ポイント

- **アプリはカーネルに騙されている**: `curl` は「繋いだ／貰った」しか知らない。実際には何十ものパケットが往復している。
- **TCPは巨大な状態機械**: シーケンス番号、ACK、再送、ウィンドウ制御… 本実習ではこの全体を自作するのは範囲を超えるため扱いません（[深掘りポイント](#-深掘りポイント)参照）。
- **本実習が目指すもの**: もっとシンプルな **ICMP（Ping）** と **UDP** を自作することで、パケットの構造とカプセル化を根本から理解する。それらができれば、TCP がやっていることも「積み重ね」として見えてきます。

#### ✅ チェックポイント

- [ ] `curl -v` の出力から、TCP 接続確立（`Connected`）の行を確認した
- [ ] `tcpdump` で `[S] [S.] [.]` の 3 パケット（3 ウェイハンドシェイク）を確認した
- [ ] `curl` の 1 行が、実際には多数のパケットで構成されていることを理解した

```bash
# 後片付け: HTTPサーバを停止（別ターミナルで Ctrl-C、または以下で kill）
pkill -f "python3 -m http.server 8000"
```

---

### STEP 1: Packet Socket (L2) - Ethernet フレームを観察・送信する

#### 目標

通常のアプリケーションは `TCP/UDP Socket` を使い、OS が L2/L3 ヘッダを生成します。自作スタックでは **AF_PACKET** を使い、Linux の通常の IP/TCP/UDP 処理とは別に Ethernet フレームを受け取り、自分でヘッダを組み立てます。NIC ドライバやカーネル全体を迂回するわけではありません。

#### 実装: C ヘッダファイル (`pkg/rawsock/rawsock.h`)

```c
#ifndef RAWSOCK_H
#define RAWSOCK_H

#ifdef __cplusplus
extern "C" {
#endif

// 指定インターフェースで Raw Socket を開く
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

```text
Regular Socket (AF_INET)
+-----------+     +------------------------+
| App | <-- | Kernel handles TCP/IP |
+-----------+     +------------------------+

Raw Socket (AF_PACKET)
+-----------+     +------------------------------------+
| App | <-- | Raw Ethernet frame (bytes) |
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

| 引数       | 値                   | 意味                           |
| :--------- | :------------------- | :----------------------------- |
| ドメイン   | `AF_PACKET`          | パケットレベルのソケット（L2） |
| タイプ     | `SOCK_RAW`           | ヘッダを含む生パケットを取得   |
| プロトコル | `ETH_P_ALL` (0x0003) | 全プロトコルタイプを受信       |

##### 他のプロトコル値の例

| プロトコル   | 値     | 説明           |
| :----------- | :----- | :------------- |
| `ETH_P_ALL`  | 0x0003 | 全パケット受信 |
| `ETH_P_IP`   | 0x0800 | IPv4 のみ受信  |
| `ETH_P_IPV6` | 0x86DD | IPv6 のみ受信  |
| `ETH_P_ARP`  | 0x0806 | ARP のみ受信   |

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

### ✅ チェックポイント

- [ ] `pkg/rawsock/rawsock.c` 内で `socket(AF_PACKET, SOCK_RAW, ...)` を呼び出している
- [ ] root 権限（sudo）なしで実行した際にエラーになることを確認した（Raw Socket の制約）

---

### STEP 2: Ethernet Frame (L2) - MAC アドレスベースの通信

#### 目標

Ethernet は物理的に隣接する機器間の通信を制御します。MAC アドレス（48bit）を使って、同一セグメント内の通信相手を特定します。

#### Ethernet II フレーム構造

```text
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
		DstMAC:    net.HardwareAddr(data[0:6]),          // 宛先 MAC
		SrcMAC:    net.HardwareAddr(data[6:12]),         // 送信元 MAC
		EtherType: binary.BigEndian.Uint16(data[12:14]), // プロトコルタイプ
		Payload:   data[14:],                            // ペイロード（上位レイヤーのデータ）
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

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Version | IHL | TOS | Total Length |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Identification | Flags | Fragment Offset |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| TTL | Protocol | Header Checksum |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Source Address |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Destination Address |
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
	// 最小ヘッダサイズ（オプション除く）
	HeaderSize = 20

	// 最大パケットサイズ
	MaxPacketSize = 65535

	// プロトコル番号
	ProtocolICMP = 1
	ProtocolTCP  = 6
	ProtocolUDP  = 17

	// フラグ
	FlagDF = 0x01 // フラグメント禁止
	FlagMF = 0x02 // まだ後続フラグメントあり
)

// Packet は IPv4 パケットを表す
type Packet struct {
	Version        uint8  // 常に 4
	IHL            uint8  // Internet Header Length（4 bytes 単位）
	TOS            uint8  // Type of Service
	TotalLength    uint16 // ヘッダ + ペイロード長
	ID             uint16 // 識別子（フラグメント再構築用）
	Flags          uint8
	FragmentOffset uint16 // フラグメントオフセット（8 bytes 単位）
	TTL            uint8  // Time to Live（ホップ数制限）
	Protocol       uint8  // 上位プロトコル
	Checksum       uint16 // ヘッダチェックサム
	SrcIP          net.IP
	DstIP          net.IP
	Options        []byte // IP オプション（通常は未使用）
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
- **チェックサム**: ヘッダの破損を検出。現実装は送信時に計算し、受信時は主に長さ・形式を検証（ヘッダチェックサム値そのものの照合は未実装）

### ✅ チェックポイント

- [ ] `Checksum` 関数が正しく実装されている（RFC 1071 準拠）
- [ ] `IHL`（ヘッダ長）が 4 バイト単位で計算されている

---

### STEP 4: ICMP Message (L4) - Ping の実装

#### 目標

ICMP (Internet Control Message Protocol) はネットワーク診断に使用されます。最も一般的な用途は「Ping」（Echo Request/Reply）です。

#### ICMP Echo メッセージ構造

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Type | Code | Checksum |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Identifier | Sequence Number |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Data |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

#### 実装: ICMP Message (`pkg/icmp/message.go`)

```go
package icmp

import (
	"encoding/binary"
	"fmt"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
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

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ethernet"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/icmp"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/rawsock"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/udp"
)

// Stack は TCP/IP スタックを表す
type Stack struct {
	sock   *rawsock.Socket
	iface  string
	srcMAC net.HardwareAddr
	srcIP  net.IP
}

// NewStack は新しいスタックを作成
func NewStack(iface string, srcIP net.IP) (*Stack, error) {
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

	// IP はカーネルに設定せず、自作スタックの応答アドレスとして渡す。
	if srcIP = srcIP.To4(); srcIP == nil {
		sock.Close()
		return nil, fmt.Errorf("source IP must be IPv4")
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
				log.Printf("recv error: %v", err)
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

// L4: プロトコル別に処理を分岐
	switch pkt.Protocol {
	case ipv4.ProtocolICMP:
		return s.handleICMP(pkt, frame.SrcMAC)
	case ipv4.ProtocolUDP:
		return s.handleUDP(pkt, frame.SrcMAC)
	default:
		return nil // TCP等は今回は無視
	}
}

// handleICMP は ICMP メッセージを処理（Ping への応答）
func (s *Stack) handleICMP(pkt *ipv4.Packet, dstMAC net.HardwareAddr) error {
	msg, err := icmp.Parse(pkt.Payload)
	if err != nil {
		return err
	}

	// Ping (Echo Request) に応答
	if msg.IsEchoRequest() && pkt.DstIP.Equal(s.srcIP) {
		log.Printf("Ping from %s: ID=%d Seq=%d", pkt.SrcIP, msg.ID, msg.Seq)
		return s.sendEchoReply(pkt, msg, dstMAC)
	}

	return nil
}

// handleUDP は UDP データグラムを処理（UDP Echo への応答）
func (s *Stack) handleUDP(pkt *ipv4.Packet, dstMAC net.HardwareAddr) error {
	msg, err := udp.Parse(pkt.Payload)
	if err != nil {
		return err
	}

	// UDP Echo (RFC 862, ポート7) への応答
	// 宛先が自分のIPであり、Echo サービスポート宛である場合のみ応答
	if pkt.DstIP.Equal(s.srcIP) && msg.DstPort == udp.EchoPort {
		log.Printf("UDP Echo from %s:%d (len=%d)", pkt.SrcIP, msg.SrcPort, len(msg.Data))
		return s.sendUDPReply(pkt, msg, dstMAC)
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

// sendUDPReply は UDP Echo Reply を送信（ICMP と並行な構造）
// 受信したデータグラムの送信元/宛先を反転し、データをそのまま返す
func (s *Stack) sendUDPReply(reqPkt *ipv4.Packet, reqMsg *udp.Message, dstMAC net.HardwareAddr) error {
	// L4: UDP Echo Reply を作成（ポートを反転、データはそのまま）
	reply := &udp.Message{
		SrcPort: reqMsg.DstPort, // 受信時の宛先ポート = 応答の送信元
		DstPort: reqMsg.SrcPort, // 受信時の送信元 = 応答の宛先
		Data:    reqMsg.Data,
	}

	// UDP データグラムをバイト列に変換（チェックサムにはIP疑似ヘッダが必要）
	udpData, err := reply.Marshal(s.srcIP, reqPkt.SrcIP)
	if err != nil {
		return fmt.Errorf("marshal udp: %w", err)
	}

	// L3: IPv4 パケットを作成
	ipPkt := &ipv4.Packet{
		Version:  4,
		IHL:      20,
		TOS:      0,
		TTL:      64,
		Protocol: ipv4.ProtocolUDP,
		SrcIP:    s.srcIP,
		DstIP:    reqPkt.SrcIP,
		Payload:  udpData,
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
	_, err = s.sock.Send(frameData)
	return err
}

// Close はスタックを閉じる
func (s *Stack) Close() error {
	return s.sock.Close()
}

func main() {
	iface := flag.String("iface", "", "Network interface to bind (required)")
	ip := flag.String("ip", "", "IPv4 address answered by this userspace stack (required)")
	flag.Parse()

	if *iface == "" || *ip == "" {
		fmt.Println("TCP/IP Protocol Stack Workshop")
		fmt.Println("Usage: sudo go run main.go -iface <interface> -ip <IPv4>")
		os.Exit(1)
	}

	stack, err := NewStack(*iface, net.ParseIP(*ip))
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

### STEP 6: UDP Message (L4) - コネクションレス通信

#### 目標

ICMP に続いて、トランスポート層のもう一つの柱 **UDP (User Datagram Protocol)** を実装します。ICMP が Ping のような「診断」用途なのに対し、UDP はアプリケーションデータを運ぶための最小限のトランスポートです。DNS、DHCP、VoIP、メトリック収集などで使われます。

UDP を自作スタックに載せることで、**「コネクション」を持たないプロトコルがいかにシンプルか**を、TCP の複雑さ（[STEP 0](#step-0-通信を観察する--カーネルが隠しているもの) のハンドシェイクを思い出してください）と対比して体感します。

#### UDP と TCP/ICMP の違い

| 観点         | ICMP       | UDP                      | TCP                           |
| :----------- | :--------- | :----------------------- | :---------------------------- |
| 接続         | なし       | なし（コネクションレス） | あり（3ウェイハンドシェイク） |
| ポート       | なし       | あり（16bit）            | あり（16bit）                 |
| 信頼性       | なし       | なし（届かなくてもOK）   | あり（再送・順序制御）        |
| チェックサム | ヘッダのみ | **疑似ヘッダ + 全体**    | 疑似ヘッダ + 全体             |
| 用途例       | Ping       | DNS, DHCP, メトリック    | HTTP, SSH, TLS                |

UDP のチェックサムが **「疑似ヘッダ (Pseudo Header)」** を使う点が重要です。送信元・宛先 IP、プロトコル番号、UDP 長も検査対象に含めるため、誤った宛先に配送されたデータグラムを検出できます。攻撃者によるなりすましを防ぐ機構ではありません。

#### UDP データグラム構造

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Source Port           |      Destination Port         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Length             |          Checksum             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                             Data                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

ヘッダは **わずか 8 バイト**（TCP ヘッダは最小 20 バイト）。シンプルさが UDP の本質です。

#### 実装: UDP Message (`pkg/udp/message.go`)

```go
package udp

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
)

const (
	// UDP ヘッダサイズ（固定）
	HeaderSize = 8

	// EchoPort は UDP Echo サービス (RFC 862) のポート番号
	EchoPort = 7
)

// Message は UDP データグラムを表す
type Message struct {
	SrcPort  uint16
	DstPort  uint16
	Length   uint16 // ヘッダ + データ長
	Checksum uint16
	Data     []byte
}

// Parse はバイト列から UDP データグラムを解析
func Parse(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("datagram too short: %d < %d", len(data), HeaderSize)
	}

	msg := &Message{
		SrcPort:  binary.BigEndian.Uint16(data[0:2]),
		DstPort:  binary.BigEndian.Uint16(data[2:4]),
		Length:   binary.BigEndian.Uint16(data[4:6]),
		Checksum: binary.BigEndian.Uint16(data[6:8]),
	}

	// ペイロードは宣言された長さから導出
// Length はヘッダを含み、IPv4 では必ず 8 以上
	payloadLen := int(msg.Length) - HeaderSize
	if payloadLen < 0 || HeaderSize+payloadLen != len(data) {
		return nil, fmt.Errorf("invalid length field: declared=%d actual=%d", msg.Length, len(data))
	}
	msg.Data = data[HeaderSize : HeaderSize+payloadLen]

	return msg, nil
}

// Marshal はデータグラムをバイト列に変換する。
// チェックサムには IPv4 の「疑似ヘッダ」が必要なため、送信元/宛先IPを受け取る。
func (m *Message) Marshal(srcIP, dstIP net.IP) ([]byte, error) {
	length := uint16(HeaderSize + len(m.Data))
	buf := make([]byte, length)

	binary.BigEndian.PutUint16(buf[0:2], m.SrcPort)
	binary.BigEndian.PutUint16(buf[2:4], m.DstPort)
	binary.BigEndian.PutUint16(buf[4:6], length)
	// Checksum は後で計算

	if len(m.Data) > 0 {
		copy(buf[HeaderSize:], m.Data)
	}

	// チェックサム計算: 疑似ヘッダ + UDPデータグラム
	// IPv4 では UDP チェックサムは必須ではないが、現代では実質必須
	// （0 の場合は「計算しない」を意味するが、Linux等は常に計算する）
	pkt := &ipv4.Packet{
		SrcIP:   srcIP,
		DstIP:   dstIP,
		Protocol: ipv4.ProtocolUDP,
		Payload: buf,
	}
	// PseudoHeader() が [SrcIP(4)][DstIP(4)][0][Proto(1)][Length(2)] = 12 バイトを返す
	// これを UDP データグラムの先頭に付けてチェックサムを計算
	pseudo := pkt.PseudoHeader()
	// 注意: PseudoHeader は payload 長を使うので、buf の Length を参照させる
	// ここでは buf が既に完全な UDP データグラムなので、payload として扱う
	checksum := ipv4.Checksum(append(pseudo, buf...))
	// チェックサム計算結果が 0x0000 の場合は 0xFFFF で送信（0 = 計算しない、と区別）
	if checksum == 0 {
		checksum = 0xFFFF
	}
	binary.BigEndian.PutUint16(buf[6:8], checksum)

	return buf, nil
}
```

#### 学習ポイント

##### 疑似ヘッダ (Pseudo Header) の役割

UDP/TCP のチェックサムは、UDP ヘッダだけではなく **IPヘッダの一部（送信元IP・宛先IP・プロトコル番号・長さ）** を含めて計算します。これを「疑似ヘッダ」と呼びます。

```text
+--------+--------+--------+--------+
|           Source IP Address       |   ← IPレイヤーの情報
+--------+--------+--------+--------+
|        Destination IP Address     |   ← IPレイヤーの情報
+--------+--------+--------+--------+
|  zero   | Protocol|  UDP Length   |   ← Protocol = 17 (UDP)
+--------+--------+--------+--------+
|         Source Port              |
+--------+--------+--------+--------+
|      (以下、実際のUDPデータグラム)  |
```

**なぜこれが必要か？** IP 層との境界で宛先やプロトコルを取り違えたデータグラムを、トランスポート層でも検出するためです。チェックサムを再計算できる攻撃者を防ぐ仕組みではありません。

##### コネクションレスの意味

UDP には ICMP と同様、**「接続確立」がありません。**

- TCP: `connect()` → `SYN/SYN-ACK/ACK` → `send()` → `recv()` → `close()` (FIN...)
- UDP: `sendto()` 一発で終わり

受信側も「誰からの要求か」をセットアップ段階で知りません。UDP データグラムが届いた瞬間に、送信元 IP・ポート（IP ヘッダと UDP ヘッダから読み取る）を知ることになります。これが STEP 5 の `handleUDP` で `pkt.SrcIP` と `msg.SrcPort` を参照して応答先を決めている理由です。

##### なぜ UDP なのか（応用）

UDP をあえて選ぶのは「少しの損よりも速さが大事」な場面です:

- **DNS**: 1 回の問い合わせに接続確立のコスト（RTT×1.5）を払いたくない
- **DHCP**: 接続確立前（IP アドレスを持たない状態）で通信する必要がある
- **メトリック/ログ収集**: 1 パケット失っても次が来ればよい
- **VoIP/動画ストリーミング**: 再送よりリアルタイム性

本実習の UDP Echo は学習用ですが、実プロトコルの土台として同じ構造が使われます。

##### TCP とのコントラスト

本教材で UDP を実装すると、TCP の「重さ」が際立ちます:

- TCP はシーケンス番号管理・ACK・再送タイマー・輻輳制御・スライディングウィンドウ・接続状態機械（11 状態）を全て実装する必要がある
- UDP は「ポート番号 + チェックサム」だけ。本ステップのコード量（約 90 行）と、TCP 実装に必要な数千行を比べてみてください

この対比こそが「なぜ DNS は UDP なのか」「なぜ HTTP/3 は UDP 上に作られたのか」を理解する鍵になります（[深掘りポイント](#-深掘りポイント)参照）。

#### ✅ チェックポイント

- [ ] `pkg/udp/message.go` で `HeaderSize = 8` を定義した
- [ ] `Marshal` が `srcIP, dstIP` を受け取り、それらを使った疑似ヘッダ込みのチェックサムを計算している
- [ ] チェックサム結果が `0x0000` の場合に `0xFFFF` に置き換えている（0 = 計算しない、との区別）
- [ ] 受信時は `Length` と、疑似ヘッダを含むチェックサムを検証してから応答する
- [ ] STEP 5 の `handlePacket` が `ProtocolICMP` / `ProtocolUDP` で分岐し、両方に応答できる

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
sudo ./tcpip_stack -iface veth-host -ip 192.168.100.1
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

### 5. UDP Echo テスト（返信本文まで検証）

UDP Echo サービス（ポート 7）にデータを送ってみます。

```bash
# Namespace 内から送信し、同じデータが返信されたことを確認
sudo ip netns exec workshop python3 - <<'PY'
import socket

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.settimeout(1)
sock.sendto(b"hello udp", ("192.168.100.1", 7))
data, peer = sock.recvfrom(1024)
assert data == b"hello udp", (data, peer)
print(f"reply={data!r} from {peer}")
PY
```

カスタムスタック側のログに `UDP Echo from 192.168.100.2:xxxxx (len=10)` のように表示されれば成功です。応答パケットは `tcpdump` で確認できます。

```bash
# 別ターミナル: UDP の往復を観察
sudo tcpdump -i veth-host -nn -vv 'udp port 7'
```

```text
13:00:05.123 IP 192.168.100.2.54321 > 192.168.100.1.7: UDP, length 10
13:00:05.124 IP 192.168.100.1.7 > 192.168.100.2.54321: UDP, length 10
```

### 6. UDP を観察して壊す

まず `tcpdump -XX` で 8 バイトの UDP ヘッダと payload を 16 進数で対応付けます。次に `pkg/udp` の単体テストで、チェックサムを 1 bit だけ反転したデータグラムが `VerifyChecksum` に拒否されることを確認します。正常系だけでなく「なぜ破棄されるか」を観察して初めて、疑似ヘッダの役割が具体化します。

---

## 📊 パケットフロー図

```text
=================== Receive Path (Rx) ===================

NIC --> Raw Socket --> Ethernet.Parse --> IPv4.Parse --> [ICMP.Parse | UDP.Parse]
                           |  |  |                  |               |
                      [MAC Filter]      [IP Filter]    [Proto Branch] [Proto Branch]
                           |  |  |                  |               |
                        Pass?              Pass?     Type=8?       Port=7?
                           |  |  |                  |               |
                           v                  v       v               v
                      Drop/Continue     Drop/Continue Drop/Reply   Drop/Reply

==================== Send Path (Tx) ====================

NIC <-- Raw Socket <-- Ethernet.Marshal <-- IPv4.Marshal <-- [ICMP.Reply | UDP.Reply]
```

**凡例:**

- `-->` : データの流れ（パケット/バイト列）
- `|` : 処理の分岐点
- `v` : 次の処理へ
- `[]` : フィルタリング処理

### パケット処理の詳細な流れ

#### 1. 受信パス（左から右へ）

Ping 要求が届いた時の処理フロー:

```text
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
・[IP Filter]: DstIP が自分宛か、Protocol が対応済みか確認
  - Protocol = 1 (ICMP) または 17 (UDP) → 継続
  - それ以外 → 破棄

ステップ 4: L4.Parse
───────────────────────────
・Protocol=1 は ICMP、Protocol=17 は UDP として解析
・UDP は Length とチェックサム（疑似ヘッダ込み）を検証
・Type=8 または DstPort=7 の場合だけ Echo Reply を生成
```

#### 2. 送信パス（右から左へ）

Echo Reply を返す時の処理フロー:

```text
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

```text
T1: Namespace 側で ping 192.168.100.1 実行
    |
    v
T2: Packet arrives (Echo Request)
    +---------------------------------------------------+
    | Ethernet: Dst=<veth-host の MAC>               |
    | IPv4:     Src=192.168.100.2, Dst=192.168.100.1 |
    | ICMP:     Type=8, ID=1234, Seq=1               |
    +---------------------------------------------------+
    |
    v  veth-host に到着
    |
T3: Custom stack receives
    | -> MAC Filter: 自分の MAC -> Pass
    | -> IP Filter:  DstIP is ours -> Pass
    | -> Type Check: Type=8 -> Generate Reply
    |
    v
T4: Packet sent (Echo Reply)
    +---------------------------------------------------+
    | Ethernet: Dst=xx:xx:xx:xx:xx:xx                |
    | IPv4:     Src=192.168.100.1, Dst=192.168.100.2 |
    | ICMP:     Type=0, ID=1234, Seq=1               |
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
2. **チェックサム**: RFC 1071 の 1 の補数和アルゴリズム。UDP/TCP では疑似ヘッダ込みで計算することで、IP レベルの改竄にも気付ける
3. **エンディアン**: ネットワークバイトオーダ (Big Endian) の重要性
4. **ICMP と UDP の対比**: どちらもコネクションレスだが、UDP はポート概念と疑似ヘッダを持つ。アプリケーションデータを運ぶ「最小限のトランスポート」としての設計
5. **TCP を実装するとしたら**: 本教材が ICMP と UDP にとどめた理由
   - **3ウェイハンドシェイク**: SYN / SYN-ACK / ACK の状態遷移（CLOSED, SYN_SENT, SYN_RECEIVED, ESTABLISHED... 全 11 状態）
   - **シーケンス番号と ACK**: 送信したバイト数を全て追跡し、相手がどこまで受信したかを把握
   - **再送タイマー (RTO)**: パケットが届かなかった場合の再送。RTT 推定には Jacobson アルゴリズム等が必要
   - **スライディングウィンドウ**: 相手の受信能力（ウィンドウサイズ）に合わせて送信量を調整
   - **輻輳制御**: Slow Start, Congestion Avoidance, Fast Retransmit（TCP Reno / Cubic / BBR など）
   - **コネクション終了**: 4 ウェイハンドシェイク（FIN/ACK/FIN/ACK）と TIME_WAIT 状態
   - 参照実装としては、教育向け軽量スタックの **lwIP**（C 言語、数千行）、C++の **Seastar**、Linux カーネルの `net/ipv4/tcp_*.c` が参考になる
6. **HTTP をしゃべるには**: 自作スタックで HTTP サーバを立てるには、まず TCP 完全実装（上記）が必要。その上で HTTP/1.1 のリクエストライン・ヘッダ・ボディのパーサを実装し、ステートマシンで Keep-Alive を管理する。HTTP/3 はさらに QUIC（UDP ベース）全体を実装する必要がある
7. **次の学習目標**: 本教材で L1〜L4 の「受動応答」（Ping/Echo）を作った後は、能動的な送信（Ping クライアント、UDP クライアント）→ TCP 実装の順に進むのが自然

## 📚 参考文献

- [RFC 791 (IPv4)](https://datatracker.ietf.org/doc/html/rfc791)
- [RFC 792 (ICMP)](https://datatracker.ietf.org/doc/html/rfc792)
- [RFC 768 (UDP)](https://datatracker.ietf.org/doc/html/rfc768)
- [RFC 862 (Echo Protocol)](https://datatracker.ietf.org/doc/html/rfc862)
- [RFC 1071 (Checksum)](https://datatracker.ietf.org/doc/html/rfc1071)
- [RFC 793 (TCP)](https://datatracker.ietf.org/doc/html/rfc793) — 本教材では実装していないが、UDP との対比を読むのに有用
- [lwIP - A Lightweight TCP/IP Stack](https://savannah.nongnu.org/projects/lwip/) — TCP 完全実装の教育用参照実装

---

## 🔧 トラブルシューティング

### Raw Socket を開けない

**症状**: `failed to open raw socket` エラー

**原因と対処**:

- root 権限で実行しているか確認

  ```bash
  sudo ./tcpip_stack -iface veth-host
  ```

- `CAP_NET_RAW` ケーパビリティがないか確認

  ```bash
  getcap ./tcpip_stack
  # なければ付与
  sudo setcap cap_net_raw+ep ./tcpip_stack
  ```

### Ping が通らない

**症状**: `From 192.168.100.2 icmp_seq=1 Destination Host Unreachable`

**原因と対処**:

- カスタムスタックが起動しているか確認

  ```bash
  sudo ip netns exec workshop ping 192.168.100.1
  ```

- インターフェースの状態を確認

  ```bash
  ip link show veth-host
  ip addr show veth-host
  ```

- tcpdump でパケットを確認

  ```bash
  sudo tcpdump -i veth-host -nn -vv icmp
  ```

### ビルドエラーが発生する

**症状**: `cgo` 関連のエラー

**原因と対処**:

- GCC がインストールされているか確認

  ```bash
  gcc --version
  # なければインストール
  sudo apt install gcc
  ```

- Linux ヘッダーがインストールされているか確認

  ```bash
  sudo apt install linux-headers-$(uname -r)
  ```

---

## 💻 環境別注意事項

### macOS の場合

**この実習は macOS では動作しません。**

理由:

- macOS は `AF_PACKET` をサポートしていません
- 代わりに `BPF (Berkeley Packet Filter)` を使用する必要があります

代替案:

- Linux VM (UTM, Parallels, VMware) を使用
- クラウド上の Ubuntu インスタンスで実行
- Docker Desktop with Linux VM

### Windows の場合

**この実習は Windows では動作しません。**

理由:

- Windows は `WinPcap/Npcap` を使用する必要があります

代替案:

- WSL2 上の Ubuntu で実行することを推奨
- クラウド上の Ubuntu インスタンスで実行

### WSL2 の場合

WSL2 は Linux カーネルベースですが、Raw Socket のサポートに制限がある場合があります。

```bash
# WSL2 のカーネルバージョンを確認
uname -r
```

問題が発生する場合は、クラウド上の Ubuntu VM での実行を推奨します。
