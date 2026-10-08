# TCP/IP プロトコルスタック実習 — パート 2: UDP・スタック統合

> **⏱️ 所要時間**: 約 80 分（全 2 パートの 2 つ目。実習全体で約 180 分）
> **🎯 レベル**: 上級
> **💻 環境**: Linux (root/sudo 必須)

TCP/IP プロトコルスタック実習のパート 2 です。[パート 1: Ethernet・IP・ICMP](tcpip_stack_ja.md) では AF_PACKET の Packet Socket と Ethernet・IPv4・ICMP の各層を構築しました。このパートでは UDP データグラムを実装し、全コンポーネントを `main.go` に統合して、自作スタックに対する Ping と UDP Echo の動作確認を行います。

> **💡 用語集**: この実習で登場する専門用語は [用語集](glossary_ja.md) を参照してください。

## ゴール

UDP レイヤー（`pkg/udp`）を実装し、全コンポーネントを `main.go` に統合して、自作スタックに実際のトラフィックで動作確認を行います。名前空間内からの Ping に `192.168.100.1` から応答が返り、UDP Echo テストでは `b'hello udp'` が自作スタックから返ることを確認します。

## 前提条件

- [ ] [パート 1: Ethernet・IP・ICMP](tcpip_stack_ja.md) を完了していること（`pkg/rawsock`、`pkg/ethernet`、`pkg/ipv4`、`pkg/icmp` がパート 1 で作成したプロジェクトに存在すること）
- [ ] パート 1 の砂場が残っていること: `ip netns list` で `workshop` が表示され、`veth-host` に IPv4 アドレスがなく、名前空間内に `192.168.100.1` の permanent ネイバーがあること（無ければパート 1 の「開発環境の準備」をやり直す）
- [ ] Go 1.22 以降と CGO が有効な GCC（パート 1 で導入済み）

---

## 🚀 実装ステップ

ステップ番号は、STEP 4 で終わったパート 1 からの通し番号です。

### STEP 5: UDP Message (L4) - コネクションレス通信

#### 目標

ICMP に続いて、トランスポート層のもう一つの柱 **UDP (User Datagram Protocol)** を実装します。ICMP が Ping のような「診断」用途なのに対し、UDP はアプリケーションデータを運ぶための最小限のトランスポートです。DNS、DHCP、VoIP、メトリック収集などで使われます。

UDP を自作スタックに載せることで、**「コネクション」を持たないプロトコルがいかにシンプルか**を、TCP の複雑さ（[STEP 0](tcpip_stack_ja.md#step-0-通信を観察する--カーネルが隠しているもの) のハンドシェイクを思い出してください）と対比して体感します。

#### UDP と TCP/ICMP の違い

| 観点         | ICMP                | UDP                      | TCP                           |
| :----------- | :------------------ | :----------------------- | :---------------------------- |
| 接続         | なし                | なし（コネクションレス） | あり（3ウェイハンドシェイク） |
| ポート       | なし                | あり（16bit）            | あり（16bit）                 |
| 信頼性       | なし                | なし（届かなくてもOK）   | あり（再送・順序制御）        |
| チェックサム | ICMP メッセージ全体 | **疑似ヘッダ + 全体**    | 疑似ヘッダ + 全体             |
| 用途例       | Ping                | DNS, DHCP, メトリック    | HTTP, SSH, TLS                |

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

[実行可能な UDP 実装](assets/tcpip_stack/pkg/udp/message.go)と[テスト](assets/tcpip_stack/pkg/udp/message_test.go)を読みます。`Parse`、`VerifyChecksum`、`Marshal` が長さの検証、チェックサム照合、応答の生成を分担します。教材内に実装の別コピーを持たず、このソースを直接使用してください。

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

受信側も「誰からの要求か」をセットアップ段階で知りません。UDP データグラムが届いた瞬間に、送信元 IP・ポート（IP ヘッダと UDP ヘッダから読み取る）を知ることになります。これが後述の STEP 6 の `handleUDP` で `pkt.SrcIP` と `msg.SrcPort` を参照して応答先を決めている理由です。

##### なぜ UDP なのか（応用）

UDP をあえて選ぶのは「少しの損よりも速さが大事」な場面です:

- **DNS**: 1 回の問い合わせにトランスポート接続確立のコストを払いたくない
- **DHCP**: 接続確立前（IP アドレスを持たない状態）で通信する必要がある
- **メトリック/ログ収集**: 1 パケット失っても次が来ればよい
- **VoIP/動画ストリーミング**: 再送よりリアルタイム性

本実習の UDP Echo は学習用ですが、実プロトコルの土台として同じ構造が使われます。

##### TCP とのコントラスト

本教材で UDP を実装すると、TCP の「重さ」が際立ちます:

- TCP はシーケンス番号管理・ACK・再送タイマー・輻輳制御・スライディングウィンドウ・接続状態機械（11 状態）を全て実装する必要がある
- UDP は「ポート番号 + チェックサム」だけ。TCP で必要になる接続状態、再送、フロー制御、輻輳制御との違いを確認してください

この対比こそが「なぜ DNS は UDP なのか」「なぜ HTTP/3 は UDP 上に作られたのか」を理解する鍵になります（[深掘りポイント](#-深掘りポイント)参照）。

#### ✅ チェックポイント

- [ ] `pkg/udp/message.go` で `HeaderSize = 8` を定義した
- [ ] `Marshal` が `srcIP, dstIP` を受け取り、それらを使った疑似ヘッダ込みのチェックサムを計算している
- [ ] チェックサム結果が `0x0000` の場合に `0xFFFF` に置き換えている（0 = 計算しない、との区別）
- [ ] 受信時は `Length` と、疑似ヘッダを含むチェックサムを検証してから応答する
- [ ] STEP 6 の `handlePacket` が `ProtocolICMP` / `ProtocolUDP` で分岐し、両方に応答できる

---

### STEP 6: Stack 統合 (`main.go`) - 全てを繋げる

#### 目標

これまでのコンポーネント（前ステップの UDP を含む）を統合し、実際に Ping と UDP Echo に応答するサーバーを構築します。

#### 実装: メインプログラム (`main.go`)

以下のリストは [assets/tcpip_stack/main.go](assets/tcpip_stack/main.go) と一致します（先頭行の `//go:build linux` ビルドタグのみ省略。本実習は Linux 専用ですが、ファイルをコピーする際はタグを残してください）。

```go
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ethernet"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/icmp"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/ipv4"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/rawsock"
	"github.com/sokoide/workshop/infra/assets/tcpip_stack/pkg/udp"
)

// Stack represents the TCP/IP stack
type Stack struct {
	sock   *rawsock.Socket
	iface  string
	srcMAC net.HardwareAddr
	srcIP  net.IP
}

// NewStack creates a new TCP/IP stack
func NewStack(iface string, srcIP net.IP) (*Stack, error) {
	sock, err := rawsock.Open(iface)
	if err != nil {
		return nil, fmt.Errorf("open raw socket: %w", err)
	}

	// Get the interface MAC. The IP address is deliberately supplied by the
	// caller, so the Linux kernel does not also own and answer for it.
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("get interface: %w", err)
	}

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

// Run starts the packet processing loop
func (s *Stack) Run(ctx context.Context) error {
	buf := make([]byte, 65536)
	log.Printf("Listening on %s (%s, MAC: %s)", s.iface, s.srcIP, s.srcMAC)
	var closeOnce sync.Once
	go func() {
		<-ctx.Done()
		closeOnce.Do(func() {
			_ = s.sock.Close()
		})
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, err := s.sock.Recv(buf)
			if err != nil {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				continue
			}

			packetData := make([]byte, n)
			copy(packetData, buf[:n])

			if err := s.handlePacket(packetData); err != nil {
				log.Printf("handle packet: %v", err)
			}
		}
	}
}

// handlePacket processes a single packet
func (s *Stack) handlePacket(data []byte) error {
	// Parse Ethernet frame
	frame, err := ethernet.Parse(data)
	if err != nil {
		return err
	}

	// Filter: Only process packets for our MAC or broadcast/multicast
	if !s.shouldProcessFrame(frame) {
		return nil
	}

	// Handle IPv4
	if frame.EtherType != ethernet.TypeIPv4 {
		return nil
	}

	pkt, err := ipv4.Parse(frame.Payload)
	if err != nil {
		return err
	}
	// Reassembly is outside this workshop's scope. Do not treat a fragment as
	// a complete ICMP message or UDP datagram.
	if pkt.IsFragmented() {
		return fmt.Errorf("fragmented IPv4 packets are unsupported")
	}

	switch pkt.Protocol {
	case ipv4.ProtocolICMP:
		return s.handleICMP(pkt, frame.SrcMAC)
	case ipv4.ProtocolUDP:
		return s.handleUDP(pkt, frame.SrcMAC)
	default:
		return nil
	}
}

func (s *Stack) handleICMP(pkt *ipv4.Packet, dstMAC net.HardwareAddr) error {
	msg, err := icmp.Parse(pkt.Payload)
	if err != nil {
		return err
	}
	if msg.IsEchoRequest() && pkt.DstIP.Equal(s.srcIP) {
		log.Printf("Ping from %s: ID=%d Seq=%d", pkt.SrcIP, msg.ID, msg.Seq)
		return s.sendEchoReply(pkt, msg, dstMAC)
	}
	return nil
}

func (s *Stack) handleUDP(pkt *ipv4.Packet, dstMAC net.HardwareAddr) error {
	msg, err := udp.Parse(pkt.Payload)
	if err != nil {
		return err
	}
	if err := udp.VerifyChecksum(pkt.Payload, pkt.SrcIP, pkt.DstIP); err != nil {
		return err
	}
	if pkt.DstIP.Equal(s.srcIP) && msg.DstPort == udp.EchoPort {
		log.Printf("UDP Echo from %s:%d (len=%d)", pkt.SrcIP, msg.SrcPort, len(msg.Data))
		return s.sendUDPReply(pkt, msg, dstMAC)
	}
	return nil
}

// shouldProcessFrame checks if we should process this frame
func (s *Stack) shouldProcessFrame(frame *ethernet.Frame) bool {
	// Process if addressed to us
	if bytes.Equal(frame.DstMAC, s.srcMAC) {
		return true
	}
	// Process if broadcast
	if ethernet.IsBroadcast(frame.DstMAC) {
		return true
	}
	// Process if multicast
	if ethernet.IsMulticast(frame.DstMAC) {
		return true
	}
	return false
}

// sendEchoReply sends an ICMP Echo Reply
func (s *Stack) sendEchoReply(reqPkt *ipv4.Packet, reqMsg *icmp.Message, dstMAC net.HardwareAddr) error {
	// Build ICMP Echo Reply
	reply := icmp.NewEchoReply(reqMsg.ID, reqMsg.Seq, reqMsg.Data)
	icmpData := reply.Marshal()

	// Build IP packet
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

	// Build Ethernet frame
	frame := &ethernet.Frame{
		DstMAC:    dstMAC,
		SrcMAC:    s.srcMAC,
		EtherType: ethernet.TypeIPv4,
		Payload:   ipData,
	}

	frameData := frame.Marshal()

	// Send
	_, err := s.sock.Send(frameData)
	return err
}

func (s *Stack) sendUDPReply(reqPkt *ipv4.Packet, reqMsg *udp.Message, dstMAC net.HardwareAddr) error {
	reply := &udp.Message{SrcPort: reqMsg.DstPort, DstPort: reqMsg.SrcPort, Data: reqMsg.Data}
	udpData, err := reply.Marshal(s.srcIP, reqPkt.SrcIP)
	if err != nil {
		return fmt.Errorf("marshal UDP: %w", err)
	}

	ipPkt := &ipv4.Packet{
		TTL: 64, Protocol: ipv4.ProtocolUDP, SrcIP: s.srcIP, DstIP: reqPkt.SrcIP, Payload: udpData,
	}
	frame := &ethernet.Frame{
		DstMAC: dstMAC, SrcMAC: s.srcMAC, EtherType: ethernet.TypeIPv4, Payload: ipPkt.Marshal(),
	}
	_, err = s.sock.Send(frame.Marshal())
	return err
}

// Close closes the stack
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
# Expected output: 64 bytes from 192.168.100.1: icmp_seq=1 ttl=64（応答 1 件につき 1 行）
```

### 4. パケット観察

```bash
sudo tcpdump -i veth-host -nn -vv icmp
# Expected output: IP 192.168.100.2 > 192.168.100.1: ICMP echo request ... / IP 192.168.100.1 > 192.168.100.2: ICMP echo reply ...
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
# Expected output: reply=b'hello udp' from ('192.168.100.1', 7)
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

## 学んだこと

- アプリケーションデータが Ethernet (L2)、IPv4 (L3)、ICMP/UDP (L4) とカプセル化される仕組みと、各ヘッダーを生バイト列から解析する方法
- Linux の `AF_PACKET` Raw Socket（CGO 経由）を使って、veth インターフェース上のフレームを観察・送信する方法
- ネットワーク名前空間と veth ペアにより、低レベルネットワーク実験用の隔離環境（砂場）を作る方法
- RFC 1071 のチェックサム（UDP 疑似ヘッダー込み）とネットワークバイトオーダ (Big Endian) がパケットの完全性をどう守るか
- TCP の完全実装（ハンドシェイク、シーケンス番号、再送制御、輻輳制御）が単独のプロジェクトになる規模であること（後述の「深掘りポイント」参照）

**次のステップ**: 実際のアプリケーションがこれらの接続をどう再利用・アップグレードするかは、[HTTP 永続接続実習](http_persistent_ja.md) を参照してください。

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

パート 1 で作成した砂場も含めて削除します:

```bash
sudo ip link delete veth-host
sudo ip netns delete workshop
```

---

## 💡 深掘りポイント

1. **ICMP と UDP の対比**: どちらもコネクションレスだが、UDP はポート概念と疑似ヘッダを持つ。アプリケーションデータを運ぶ「最小限のトランスポート」としての設計
2. **TCP を実装するとしたら**: 本教材が ICMP と UDP にとどめた理由
   - **3ウェイハンドシェイク**: SYN / SYN-ACK / ACK の状態遷移（CLOSED, SYN_SENT, SYN_RECEIVED, ESTABLISHED... 全 11 状態）
   - **シーケンス番号と ACK**: 送信したバイト数を全て追跡し、相手がどこまで受信したかを把握
   - **再送タイマー (RTO)**: パケットが届かなかった場合の再送。RTT 推定には Jacobson アルゴリズム等が必要
   - **スライディングウィンドウ**: 相手の受信能力（ウィンドウサイズ）に合わせて送信量を調整
   - **輻輳制御**: Slow Start, Congestion Avoidance, Fast Retransmit（TCP Reno / Cubic / BBR など）
   - **コネクション終了**: 4 ウェイハンドシェイク（FIN/ACK/FIN/ACK）と TIME_WAIT 状態
   - 参照実装としては、教育向け軽量スタックの **lwIP**（C 言語、数千行）、C++の **Seastar**、Linux カーネルの `net/ipv4/tcp_*.c` が参考になる
3. **HTTP をしゃべるには**: 自作スタックで HTTP サーバを立てるには、まず TCP 完全実装（上記）が必要。その上で HTTP/1.1 のリクエストライン・ヘッダ・ボディのパーサを実装し、ステートマシンで Keep-Alive を管理する。HTTP/3 はさらに QUIC（UDP ベース）全体を実装する必要がある
4. **次の学習目標**: 本教材で L1〜L4 の「受動応答」（Ping/Echo）を作った後は、能動的な送信（Ping クライアント、UDP クライアント）→ TCP 実装の順に進むのが自然

## 📚 参考文献

- [RFC 768 (UDP)](https://datatracker.ietf.org/doc/html/rfc768)
- [RFC 862 (Echo Protocol)](https://datatracker.ietf.org/doc/html/rfc862)
- [RFC 793 (TCP)](https://datatracker.ietf.org/doc/html/rfc793) — 本教材では実装していないが、UDP との対比を読むのに有用
- [lwIP - A Lightweight TCP/IP Stack](https://savannah.nongnu.org/projects/lwip/) — TCP 完全実装の教育用参照実装

---

## 🔧 トラブルシューティング

Raw Socket のオープンエラーやビルドエラーについては、[パート 1](tcpip_stack_ja.md) のトラブルシューティングを参照してください。

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
