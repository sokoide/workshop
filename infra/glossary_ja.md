# インフラ勉強会用語集

このドキュメントは、各ワークショップで登場する専門用語の簡潔な説明をまとめたものです。

## <a name="network"></a>ネットワーク用語

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

### IP アドレス

Internet Protocol Address。ネットワーク上のデバイスを識別するためのアドレス。IPv4 は 32bit、IPv6 は 128bit。

### サブネットマスク

IP アドレスのどの部分がネットワーク部で、どの部分がホスト部かを示す。

### デフォルトゲートウェイ

自サブネット外の宛先へのパケットを転送するルーターの IP アドレス。

### NAT (Network Address Translation)

プライベート IP アドレスとグローバル IP アドレスを変換する技術。

---

## <a name="container"></a>コンテナ・仮想化用語

### Rootful / Rootless

- **Rootful**: root 権限で動作するコンテナランタイム。
- **Rootless**: 一般ユーザー権限で動作するコンテナランタイム。

### Network Namespace

Linux カーネルの機能で、ネットワークスタック（インターフェース、ルーティングテーブル、ファイアウォールルール）をプロセスごとに分離する。

### veth (Virtual Ethernet)

仮想のイーサネットケーブルペア。Network Namespace 間の接続に使用。

---

## <a name="kubernetes"></a>Kubernetes 用語

### Pod

Kubernetes におけるデプロイの最小単位。同じネットワーク名前空間、ストレージ、IP アドレスを共有する 1 つ以上のコンテナで構成される。

### Deployment

Pod の「あるべき状態」を管理するリソース。レプリケーション、自己修復、ローリングアップデートなどを担当する。

### Service

Pod の集合に対する論理的な名前と、それらにアクセスするためのポリシー（固定 IP や DNS）を定義する抽象化レイヤー。

### ConfigMap / Secret

アプリケーションの設定データや機密情報（パスワード、鍵など）を、コンテナイメージから分離して Pod に注入するためのリソース。

### PersistentVolume (PV) / PersistentVolumeClaim (PVC)

K8s のストレージ抽象化。PV は実際のストレージリソース、PVC はユーザーや Pod によるストレージの利用要求。

### Helm

Kubernetes 用のパッケージマネージャー。複雑なアプリケーションを「Chart」として定義・インストール・管理できる。

### Ingress

クラスタ内のサービスへの外部アクセス（主に HTTP/HTTPS）を管理する API オブジェクト。

---

## <a name="security"></a>セキュリティ・ゼロトラスト用語

### Zero Trust Architecture (ZTA)

「決して信頼せず、常に検証する」という原則に基づくセキュリティモデル。アクセス元の場所に関わらず、すべてのアクセス要求に対して認証と認可を行う。

### mTLS (Mutual TLS)

相互 TLS。セキュアな接続を確立する前に、クライアントとサーバーの両方がデジタル証明書を使用して互いの身元を証明するプロセス。

### SPIFFE / SPIRE

- **SPIFFE**: Secure Production Identity Framework for Everyone。ソフトウェアサービスを識別するためのオープン標準。
- **SPIRE**: SPIFFE Runtime Environment。SPIFFE API を実装し、ワークロードに対して ID を発行するランタイム。

### SVID (SPIFFE Verifiable Identity Document)

ワークロードが他のワークロードに対して自分の身元を証明するために使用する文書（通常は X.509 証明書）。

### RBAC (Role-Based Access Control)

役割ベースのアクセス制御。組織内での個々のユーザーの役割に基づいて、リソースへのアクセスを制限する方法。

### NetworkPolicy

Pod 間の通信をネットワークレベル（L3/L4）で制御するための Kubernetes リソース。

---

## <a name="mesh"></a>サービスメッシュ用語

### Service Mesh

サービス間通信を管理するための専用インフラ層。可観測性、セキュリティ、信頼性などの機能を提供する。

### Sidecar Proxy (Envoy)

アプリケーションコンテナと同じ Pod 内にデプロイされるプロキシインスタンス。すべての送受信トラフィックをインターセプトして管理する。

### Istio

マイクロサービスを保護、接続、監視するためのオープンソースのサービスメッシュ。

### Control Plane / Data Plane

- **Control Plane**: プロキシ（Envoy）の管理と設定を行う（例：Istiod）。
- **Data Plane**: サービス間の実際のトラフィックを処理する（例：Envoy プロキシ）。

---

## OAuth2・アイデンティティ用語

### OAuth2

認可のための標準プロトコル。ユーザーのパスワードを教えることなく、サードパーティアプリケーションにリソースへのアクセスを許可する。

### Access Token (JWT)

特定のリソースへのアクセス権を証明するためのトークン。JSON Web Token (JWT) 形式が一般的。

### Entra ID (Azure AD)

Microsoft が提供するクラウドベースのアイデンティティおよびアクセス管理サービス。

---

## メッセージキュー用語

### Pub/Sub (Publish/Subscribe)

非同期通信パターンのひとつ。送信者（Publisher）がメッセージを送り、受信者（Subscriber）がそれを購読する。

### Exchange (RabbitMQ)

メッセージをルール（Direct, Fanout, Topic, Headers）に基づいてキューにルーティングするコンポーネント。

---

## キャッシュ用語

### Sorted Set (ZSET)

Redis のデータ構造。スコアによってソートされたメンバーの集合。ランキングの実装などに適している。

### Write-Through / Cache-Aside

- **Write-Through**: キャッシュとバックエンドデータベースの両方に同時にデータを書き込む方式。
- **Cache-Aside**: まずキャッシュを確認し、データがなければ（キャッシュミス）データベースから読み取ってキャッシュを更新する方式。

---

## プロトコル用語

### TCP/IP Stack

TCP と IP を中心とした、インターネット通信プロトコルの階層的な実装。

### Raw Socket

OS のネットワークスタックをバイパスして、生のパケットを送受信するためのソケット。

---

## Go 言語用語

### Clean Architecture

関心を同心円状のレイヤー（Domain, UseCase, Infra, Framework）に分離するソフトウェア設計パターン。

### CGO

Go から C 言語のコードを呼び出すための仕組み。
