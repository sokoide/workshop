# TCP/IP プロトコルスタック実習 — パート 1: Ethernet・IP・ICMP

> **⏱️ 所要時間**: 約 100 分（全 2 パートの 1 つ目。実習全体で約 180 分）
> **🎯 レベル**: 上級
> **💻 環境**: Linux (root/sudo 必須)

この実習（全 2 パート）では、Go と C を組み合わせて **TCP/IP プロトコルスタックをゼロから構築** します。OS カーネルが隠蔽している「パケットの生成・分解」を自らの手で実装することで、ネットワークの動作原理を抽象概念ではなく、具体的なデータ構造として理解することを目標とします。パート 1 では観察ステップと Packet Socket・Ethernet・IPv4・ICMP の各層を扱います。[パート 2: UDP・スタック統合](tcpip_stack_udp_ja.md) で UDP データグラムを実装し、`main.go` に統合して、Ping／UDP Echo の動作確認まで進みます。

> **💡 用語集**: この実習で登場する専門用語は [用語集](glossary_ja.md) を参照してください。

## ゴール

カーネルが普段隠している通信の中身を観察し、それを自分の手で実装します。このパートを終える時点で、AF_PACKET の Packet Socket 上に `pkg/rawsock`・`pkg/ethernet`・`pkg/ipv4`・`pkg/icmp` の各パッケージが構築済みとなり、`go build ./...` でスタック全体がコンパイルできる状態になることを確認します。スタックの統合と Ping／UDP Echo の動作確認は[パート 2](tcpip_stack_udp_ja.md)で行います。

## 前提条件

- [ ] Linux (Ubuntu 22.04/24.04 推奨) — Raw Socket (`AF_PACKET`) とネットワーク名前空間が必須
- [ ] Go 1.22 以降（`go version`）
- [ ] CGO が有効な GCC（`gcc --version`、`go env CGO_ENABLED`）
- [ ] `tcpdump` と `iproute2`（`ip`、`ip netns`）— `sudo apt install -y golang-go gcc tcpdump iproute2` で導入
- [ ] root/sudo 権限（ネットワーク名前空間の作成と Raw Socket のオープンには `CAP_NET_RAW` が必要）

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

パート 1 では上から 4 行（Packet Socket 〜 ICMP Message）を実装します。UDP と統合 `main.go` スタックは[パート 2](tcpip_stack_udp_ja.md) で扱います。

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
# Expected output: Flags [S], [S.]（3ウェイハンドシェイク）→ [P.]（HTTP データ）→ [F]/[R]（切断）の順にパケットが表示される
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

[実行可能な IPv4 実装](assets/tcpip_stack/pkg/ipv4/packet.go)と[テスト](assets/tcpip_stack/pkg/ipv4/packet_test.go)を読みます。`Parse` はバージョン・IHL・全長・ヘッダチェックサムを検証してから各フィールドを抽出し、`Marshal` はヘッダの組み立て・ペイロードのコピー・チェックサム計算を行います。教材内に実装の別コピーを持たず、このソースを直接使用してください。

参照実装をプロジェクトにコピーします（ワークショップリポジトリルート＝`tcpip_stack/` の親ディレクトリで実行）:

```bash
cp assets/tcpip_stack/pkg/ipv4/packet.go assets/tcpip_stack/pkg/ipv4/packet_test.go tcpip_stack/pkg/ipv4/
```

このモジュールの中核は RFC 1071 の 1 の補数チェックサムです:

```go
// Checksum calculates the IPv4 header checksum
func Checksum(data []byte) uint16 {
	sum := uint32(0)

	// Sum all 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Handle odd length
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}
```

`PseudoHeader` は、[パート 2](tcpip_stack_udp_ja.md) のトランスポート層チェックサム（UDP の疑似ヘッダ）が対象とする 12 バイトのプレフィックス（送信元 IP・宛先 IP・ゼロ・プロトコル・長さ）を生成します。

#### 学習ポイント

- **IHL (Internet Header Length)**: ヘッダ長を 4 bytes 単位で表現（通常は 5 = 20 bytes）
- **TTL**: ループ防止用。ルータを通過するたびに減算され、0 で破棄
- **チェックサム**: ヘッダの破損を検出。実装は送信時に計算し、受信時に IHL・全長とともに検証する

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

[実行可能な ICMP 実装](assets/tcpip_stack/pkg/icmp/message.go)と[テスト](assets/tcpip_stack/pkg/icmp/message_test.go)を読みます。`Parse` はチェックサムを検証してから Type/Code/ID/Seq を抽出し、`NewEchoRequest`/`NewEchoReply` が Ping の要求・応答を組み立てます。エラー系のコンストラクタ（`NewDestinationUnreachable`、`NewTimeExceeded`）は元の IPv4 ヘッダと先頭最大 8 バイトのペイロードを引用します。教材内に実装の別コピーを持たず、このソースを直接使用してください。

参照実装をプロジェクトにコピーします（ワークショップリポジトリルートで実行）:

```bash
cp assets/tcpip_stack/pkg/icmp/message.go assets/tcpip_stack/pkg/icmp/message_test.go tcpip_stack/pkg/icmp/
```

#### 学習ポイント

- **Type 8/0**: Echo Request (8) に対して Echo Reply (0) を返す
- **ID/Seq**: 複数の Ping を区別するための識別子
- **Data**: 往復時間（RTT）計測用のタイムスタンプなどを格納

## 🧪 動作確認

パート 1 では、実装したパッケージがコンパイルできることを確認します。スタック全体のビルドと Ping／UDP Echo の動作確認は[パート 2](tcpip_stack_udp_ja.md) で行います。

### 全パッケージのビルド

```bash
go build ./...
# Expected output: 出力なし — 全パッケージのコンパイル成功
```

パッケージ作成時にアセットのテストファイルもコピーした場合は、`go test ./...` で実行できます。

---

## 学んだこと

- アプリケーションデータが Ethernet (L2)、IPv4 (L3) とカプセル化される仕組みと、各ヘッダーを生バイト列から解析する方法
- Linux の `AF_PACKET` Raw Socket（CGO 経由）を使って、veth インターフェース上のフレームを観察・送信する方法
- ネットワーク名前空間と veth ペアにより、低レベルネットワーク実験用の隔離環境（砂場）を作る方法
- RFC 1071 のチェックサムとネットワークバイトオーダ (Big Endian) がパケットの完全性をどう守るか

**次のステップ**: [パート 2: UDP・スタック統合](tcpip_stack_udp_ja.md) で UDP データグラムを実装し、全コンポーネントを `main.go` に統合して、Ping と UDP Echo の動作確認を行います。

---

## 🧹 後片付け

[パート 2](tcpip_stack_udp_ja.md) へそのまま進む場合は、次のパートで砂場を再利用するため後片付けは不要です。削除する場合は以下を実行します:

```bash
sudo ip link delete veth-host
sudo ip netns delete workshop
```

---

## 💡 深掘りポイント

1. **Zero-Copy**: CGO で `unsafe.Pointer` を使いコピーを回避
2. **チェックサム**: RFC 1071 の 1 の補数和アルゴリズム。UDP/TCP では疑似ヘッダ込みで計算することで、IP レベルの改竄にも気付ける
3. **エンディアン**: ネットワークバイトオーダ (Big Endian) の重要性

## 📚 参考文献

- [RFC 791 (IPv4)](https://datatracker.ietf.org/doc/html/rfc791)
- [RFC 792 (ICMP)](https://datatracker.ietf.org/doc/html/rfc792)
- [RFC 1071 (Checksum)](https://datatracker.ietf.org/doc/html/rfc1071)

---

## 🔧 トラブルシューティング

### Raw Socket を開けない

**症状**: `failed to open raw socket` エラー

**原因と対処**:

- root 権限で実行しているか確認

  ```bash
  sudo ./tcpip_stack -iface veth-host -ip 192.168.100.1
  ```

- `CAP_NET_RAW` ケーパビリティがないか確認

  ```bash
  getcap ./tcpip_stack
  # なければ付与
  sudo setcap cap_net_raw+ep ./tcpip_stack
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

---

## 💻 環境別注意事項

### macOS の場合

**Raw Socket を使うアプリケーションの実行には Linux が必要です。** 純粋なパケット解析のテストは macOS でも実行できます。

理由:

- macOS は `AF_PACKET` をサポートしていません
- 代わりに `BPF (Berkeley Packet Filter)` を使用する必要があります

代替案:

- Linux VM (UTM, Parallels, VMware) を使用
- クラウド上の Ubuntu インスタンスで実行
- Docker Desktop with Linux VM

### Windows の場合

**Raw Socket を使うアプリケーションの実行には Linux が必要です。** 純粋な Go のパケット解析テストには Raw Socket は不要です。

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
