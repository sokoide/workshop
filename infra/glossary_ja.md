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
```

### IP アドレス

Internet Protocol Address。ネットワーク上のデバイスを識別するためのアドレス。IPv4 は 32bit、IPv6 は 128bit。

```text
IPv4 例: 192.168.1.1
IPv6 例: 2001:db8::1
```

### サブネットマスク

IP アドレスのどの部分がネットワーク部で、どの部分がホスト部かを示す。

```text
例: 255.255.255.0 (/24)
     -> 192.168.1.0 〜 192.168.1.255 の範囲
```

### デフォルトゲートウェイ

自サブネット外の宛先へのパケットを転送するルーターの IP アドレス。

### NAT (Network Address Translation)

プライベート IP アドレスとグローバル IP アドレスを変換する技術。

```text
プライベート: 192.168.1.10 ---[NAT]---> グローバル: 203.0.113.5
```

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
```

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
```

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
```

### Sets

重複を許さないメンバーの集合。

```text
SADD banned_users player2
SISMEMBER banned_users player2  # 1 (true)
```

---

## プロトコル用語

### TCP/IP スタック

TCP と IP を中心とするインターネット通信プロトコルの階層実装。

### カプセル化

上位レイヤーのデータを下位レイヤーのペイロードとして包み込む処理。

```text
[ICMP] → [IPv4 [ICMP]] → [Ethernet [IPv4 [ICMP]]]
```

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
```

### Context (context.Context)

キャンセル信号、タイムアウト、期限付き値を伝播する Go の標準パターン。

### インターフェース (interface)

メソッドの集合を定義した型。「Accept interfaces, return structs」が原則。

### CGO

Go から C コードを呼び出すための仕組み。`import "C"` で有効化。

### unsafe.Pointer

Go の型システムの制約をバイパスするポインター型。C との連携で使用。
