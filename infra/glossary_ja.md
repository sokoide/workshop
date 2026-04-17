# インフラ勉強会用語集

このドキュメントは、各ワークショップで登場する専門用語の簡潔な説明をまとめたものです。

## ネットワーク用語

### L1 / L2 / L3 (Layer 1/2/3)

OSI 参照モデルまたは TCP/IP モデルのレイヤーを指します。

| レイヤー | 名称 | 主なプロトコル/機器 | 説明 |
| :--- | :--- | :--- | :--- |
| L1 | 物理層 | ケーブル、NIC | 電気信号の送受信 |
| L2 | データリンク層 | Ethernet, MAC アドレス | 同一セグメント内の通信 |
| L3 | ネットワーク層 | IP, ルーター | 異なるネットワーク間のルーティング |
| L4 | トランスポート層 | TCP, UDP | エンドツーエンドの通信制御 |

### L2 分離 (Layer 2 Separation)

同じ物理ネットワーク上で、論理的に通信を分離する技術。VLAN や VxLAN などが該当します。

### L3 ルーティング (Layer 3 Routing)

異なる IP サブネット間でパケットを転送する機能。ルーターが担当します。

### MAC アドレス

Media Access Control Address。ネットワークインターフェース（NIC）に割り当てられる 48bit の一意識別子。

```text
例: 00:1A:2B:3C:4D:5E
```text

### IP アドレス

Internet Protocol Address。ネットワーク上のデバイスを識別するためのアドレス。IPv4 は 32bit、IPv6 は 128bit。

```text
IPv4 例: 192.168.1.1
IPv6 例: 2001:db8::1
```text

### サブネットマスク

IP アドレスのどの部分がネットワーク部で、どの部分がホスト部かを示す。

```text
例: 255.255.255.0 (/24)
     -> 192.168.1.0 〜 192.168.1.255 の範囲
```text

### デフォルトゲートウェイ

自サブネット外の宛先へのパケットを転送するルーターの IP アドレス。

### NAT (Network Address Translation)

プライベート IP アドレスとグローバル IP アドレスを変換する技術。

```text
プライベート: 192.168.1.10 ---[NAT]---> グローバル: 203.0.113.5
```text

---

## コンテナ・仮想化用語

### Rootful / Rootless

- **Rootful**: root 権限で動作するコンテナランタイム
- **Rootless**: 一般ユーザー権限で動作するコンテナランタイム

### Network Namespace

Linux カーネルの機能で、ネットワークスタック（インターフェース、ルーティングテーブル、ファイアウォールルール）をプロセスごとに分離する。

### veth (Virtual Ethernet)

仮想のイーサネットケーブルペア。Network Namespace 間の接続に使用。

```text
┌─────────────┐     veth      ┌─────────────┐
│ Namespace A │<------------->│ Namespace B │
└─────────────┘   (peer)      └─────────────┘
```text

---

## セキュリティ用語

### TLS/SSL

Transport Layer Security / Secure Sockets Layer。通信を暗号化するプロトコル。現在は TLS が主流。

### CA (Certificate Authority)

認証局。デジタル証明書を発行する信頼された第三者機関。

### 証明書チェーン

ルート CA → 中間 CA → サーバー証明書という階層構造。

### SAN (Subject Alternative Name)

証明書で有効なドメイン名を指定する拡張機能。モダンなブラウザでは CN より SAN が優先されます。

### CSR (Certificate Signing Request)

証明書署名要求。サーバーの公開鍵と情報を含み、CA に署名を依頼するファイル。

### OAuth2

認可（Authorization）のための標準プロトコル。サードパーティアプリケーションに、ユーザーのパスワードを教えることなくリソースへのアクセス権を付与する仕組み。

### Access Token

リソースサーバーに対して、特定のリソースへのアクセス権を証明するためのトークン。通常は JWT (JSON Web Token) 形式が使われます。

### Resource Server

保護されたユーザーリソースを保持するサーバー。アクセストークンを検証し、正当なリクエストに対してリソースを提供します。

---

## メッセージキュー用語

### Pub/Sub (Publish/Subscribe)

送信者（Publisher）がメッセージを送信し、受信者（Subscriber）が購読する非同期通信パターン。

### Exchange

メッセージのルーティングを行うコンポーネント。RabbitMQ では複数のタイプが存在。

| タイプ | 説明 |
| :--- | :--- |
| Direct | 完全一致のルーティングキー |
| Fanout | 全てのキューに配信 |
| Topic | ワイルドカードパターンマッチ |
| Headers | ヘッダー属性でのマッチ |

### Routing Key

メッセージの宛先を指定するためのキー。ドット区切りの形式が一般的。

```text
例: market.btc.usd
```text

### ワイルドカード

Topic Exchange で使用するパターンマッチ文字。

| 文字 | 説明 | 例 |
| :--- | :--- | :--- |
| `*` | 1 単語にマッチ | `market.*.jpy` → `market.btc.jpy` にマッチ |
| `#` | 0 個以上の単語にマッチ | `market.#` → `market.btc.usd` にマッチ |

---

## キャッシュ用語

### Sorted Set (ZSET)

Redis のデータ構造。スコア順にソートされたメンバーの集合。

```text
ZADD game_leaderboard 100 player1
ZADD game_leaderboard 250 player2
ZREVRANGE game_leaderboard 0 9  # 上位 10 名を取得
```text

### Sets

重複を許さないメンバーの集合。

```text
SADD banned_users player2
SISMEMBER banned_users player2  # 1 (true)
```text

---

## ロードバランサ用語

### Load Balancer (ロードバランサ)

複数のサーバーにトラフィックを分散する装置またはソフトウェア。単一障害点（SPOF）を回避し、可用性とスケーラビリティを向上させる。

### VIP (Virtual IP)

仮想 IP アドレス。ロードバランサーがクライアントからのリクエストを受け付けるための不変の IP アドレス。

### DNAT (Destination NAT)

宛先 IP アドレスを書き換える技術。VIP 宛のパケットをバックエンドサーバーの IP に変換する。

```text
192.168.10.100:80 (VIP) → 192.168.20.10:80 (Backend)
```text

### SNAT (Source NAT)

送信元 IP アドレスを書き換える技術。バックエンドからのレスポンスが正しくロードバランサー経由で戻るようにする。

### Round Robin (ラウンドロビン)

リクエストを順番に各サーバーに割り振るスケジューリングアルゴリズム。

### Least Connections (最小接続数)

最も接続数が少ないサーバーにリクエストを割り振るスケジューリングアルゴリズム。

### ipvs (IP Virtual Server)

Linux カーネルレベルのロードバランシング機能。

### DSR (Direct Server Return)

バックエンドサーバーからクライアントに直接応答を返すモード。ロードバランサーはリクエストのみを転送し、戻りパケットを処理しないため、スループットが向上する。

---

## レート制限用語

### Rate Limiting (レート制限)

一定期間内のリクエスト数を制限する技術。サービスの過負荷を防ぎ、公平性を保つ。

### Fixed Window (固定窓)

時間を固定枠に区切り、各枠内のリクエスト数をカウントするアルゴリズム。実装が簡単だが、枠の境界でスパイクが発生する可能性がある。

### Sliding Window (スライディング窓)

直近 N 秒間のリクエスト数を正確にカウントするアルゴリズム。滑らかな制限が可能だが、計算コストが高い。

### Token Bucket (トークンバケット)

一定レートで補充されるトークンバケットからトークンを消費するアルゴリズム。バースト許容と定常レート制限のバランスが取れている。

```text
[容量: 10] [補充レート: 1/秒]
リクエストごとに1トークン消費
```text

### Throttling (スロットリング)

リクエストの頻度を制限すること。帯域幅制限や API 呼び出し制限などで使用。

---

## 冪等性用語

### Idempotency (冪等性)

同じ操作を何度行っても結果が変わらない性質。

```text
例: x = 5 （冪等）
例: x++ （非冪等）
```text

### Idempotency Key (冪等性キー)

クライアントが生成する一意識別子。再試行時に同じキーを使用することで、重複処理を防ぐ。

```text
Header: Idempotency-Key: uuid-v4
```text

### Retry (再試行)

失敗したリクエストを再送すること。冪等性がない操作では危険。

### Dead Letter Queue (デッドレターキュー)

処理に失敗したメッセージを一時的に格納するキュー。後で再試行 or 分析に使用。

---

## プロトコル用語

### TCP/IP スタック

TCP と IP を中心とするインターネット通信プロトコルの階層実装。

### カプセル化

上位レイヤーのデータを下位レイヤーのペイロードとして包み込む処理。

```text
[ICMP] → [IPv4 [ICMP]] → [Ethernet [IPv4 [ICMP]]]
```text

### Raw Socket

OS のネットワークスタックをバイパスし、生のパケットを送受信するソケット。

### プロミスキャスモード

自分宛てでない全てのパケットを受信するモード。パケット解析ツール等で使用。

### チェックサム

データの破損を検出するための検証値。RFC 1071 では「1 の補数和の 1 の補数」を使用。

### MTU (Maximum Transmission Unit)

1 回で送信できる最大データサイズ。Ethernet では通常 1500 バイト。

---

## ポッドマン (Podman) 用語

### Podman

Docker と互換性のあるコンテナエンジン。デーモンレスで rootless 実行が可能。

### Pod

1 つ以上のコンテナをグループ化した単位。同じネットワーク名前空間を共有。

---

## Go 言語用語

### Clean Architecture

依存関係を内側に向ける設計パターン。

```text
Framework → UseCase → Domain ← Infra
            (depends on)
```text

### Context (context.Context)

キャンセル信号、タイムアウト、期限付き値を伝播する Go の標準パターン。

### インターフェース (interface)

メソッドの集合を定義した型。「Accept interfaces, return structs」が原則。

### CGO

Go から C コードを呼び出すための仕組み。`import "C"` で有効化。

### unsafe.Pointer

Go の型システムの制約をバイパスするポインター型。C との連携で使用。
