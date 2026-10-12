# ワークショップリポジトリ

---

🌐 Available languages:
[English](./README.md) | [日本語](./README_ja.md)

---

このリポジトリは、さまざまな技術トピックに関する実習用の資料を保管します。

この資料集はソフトウェアエンジニア向けのハンズオン集です。各ワークショップは独立しているため、下記の各難易度セクション内の推奨順に従っても、興味のあるトピックから直接始めても構いません。難易度ラベルは各ドキュメントの 🎯 Level 表記に基づき、前提となるワークショップは一覧内に注記しています。

## 目的から学習経路を選ぶ

各教材はリポジトリルートから準備します。必要なツールと権限は教材の前提条件を確認してください。Go は各アセットの `go.mod` に記載したバージョン以上を使います。古い Go の自動ツールチェーン取得に頼る場合は、ネットワーク接続と `GOTOOLCHAIN` 設定も確認します。

| 目的                | 前提と環境                                            | 最初の成果と経路                                                                                                                                                          |
| ------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go の設計を理解する | 基本的な Go、Go 1.23+、Python 3、curl。コンテナ不要   | [WS8](software/clean_arch_ws8_ja.md) で挨拶アプリを変更 → [解説](software/clean_arch_ja.md) → [WS1](software/clean_arch_ws1_ja.md) → [WS2](software/clean_arch_ws2_ja.md) |
| 外部サービスを使う  | 基本的な Go、Podman、各教材の Go 要件                 | [Redis](infra/cache_ja.md) でランキング表示 → [RabbitMQ](infra/rabbitmq_ja.md) → [冪等性](software/idempotency_ja.md)                                                     |
| Kubernetes を使う   | コンテナの基礎、kind・kubectl・Helm、Linux/macOS/WSL2 | [基礎](infra/kubernetes_basics_ja.md) の 3 セッションでアプリを公開 → [運用](infra/kubernetes_operations_ja.md)                                                           |
| 通信の内部を調べる  | Linux、sudo、C コンパイラー、Go。上級の独立経路       | [TCP/IP 前編](infra/tcpip_stack_ja.md) でフレーム受信 → [UDP・統合](infra/tcpip_stack_udp_ja.md)                                                                          |

BBS の WS3〜WS7 は [開始状態生成ツール](tools/prepare_bbs.py) で別々のコピーを作ります。WS3 の gRPC コードを WS4 へ持ち越す必要はありません。カタログは必要なパターンから直接読めます。

## 内容

各セクションは難易度別に分類し、レベル内では推奨順に並べています。まずはソフトウェア・アーキテクチャの初級セクションにある WS8 から始めてください。

### インフラ関連

#### 初級

- [レート制限実習：Redis で作るリクエスト制限](./infra/rate_limiting_ja.md) — **初級**
  - Fixed Window、Sliding Window、Token Bucket アルゴリズムを Redis で実装

#### 中級

- [Redis 実習：Sorted Sets で作るリアルタイム・ゲームランキング](./infra/cache_ja.md) — **中級**
  - Redis の Sorted Sets を活用した高速なランキングシステム構築と、Sets による不正ユーザー管理を学習
- [TLS/SSL 証明書実習：自己署名 CA 構築と証明書チェーンを理解する](./infra/tls_ja.md) — **中級**
  - OpenSSL で CA 構築 → 証明書の検証までの一連の流れを体験
- [リバースプロキシ実習：Traefik で学ぶ K8s Ingress の裏側](./infra/traefik_ja.md) — **中級**
  - Traefik + 複数バックエンドコンテナで SSL 終端・ルーティング・ヘルスチェック。推奨: その後に HTTPS を追加する場合は [TLS 実習](./infra/tls_ja.md)
- [Podman 実習：macvlan + ルータコンテナで L3 分離ネットワークを作る](./infra/vlan_ja.md) — **中級**
  - VLAN サブインタフェース上に macvlan ネットワークを作成し、ルータコンテナで NAT/転送
- [CoreDNS 実習：親子 DNS サーバーを構築して名前解決を理解する](./infra/coredns_ja.md) — **中級**
  - 権威サーバーとフォワーディングを 2 台の VM で構築し、ゾーン委譲の仕組みを体験。推奨: DNS クエリが UDP データグラムとして流れる様子は [TCP/IP スタック パート 2](./infra/tcpip_stack_udp_ja.md)
- [Load Balancer 実習：iptables と IPVS で作る仮想ロードバランサ](./infra/load_balancer_ja.md) — **中級**
  - iptables/IPVS を用いたロードバランシングアルゴリズム、NAT/DSR モード、ヘルスチェックを学習。任意の深掘り: [TCP/IP スタック実習](./infra/tcpip_stack_ja.md)（NAT/DNAT の基礎）
- [HTTP 永続接続実習：Keep-Alive, WebSocket, HTTP/2, gRPC, HTTP/3](./infra/http_persistent_ja.md) — **中級**
  - HTTP/1.1 から最新の HTTP/3 (QUIC) まで、接続再利用と 0-RTT の仕組みを比較。任意の深掘り: [TCP/IP スタック実習](./infra/tcpip_stack_ja.md)（TCP ハンドシェイクと接続ライフサイクル）
- [Kubernetes 実習：コンテナオーケストレーションの基礎から実践まで](./infra/kubernetes_basics_ja.md) — **中級**
  - Pod, Deployment, Service, ConfigMap, Helm 等、K8s の基本リソースとエコシステムを網羅的に学習。修了後は [Kubernetes 運用・セキュリティ実習](./infra/kubernetes_operations_ja.md) へ
- [オブジェクトストレージ実習：MinIO で学ぶ S3 互換 API](./infra/minio_ja.md) — **中級**
  - MinIO 構築 + AWS CLI/SDK でバケット操作・署名付き URL 生成
- [シークレット管理実習：HashiCorp Vault による安全な API キー管理](./infra/secret_management_ja.md) — **中級**
  - Vault KV v2 を活用したシークレットの集中管理・バージョニング・Clean Architecture による統合
- [メッセージキュー実習：RabbitMQ で学ぶ Pub/Sub とキューイング](./infra/rabbitmq_ja.md) — **中級**
  - RabbitMQ の Topic Exchange を活用したリアルタイム仮想通貨監視システム構築
- [サービスメッシュ基礎実習：Envoy サイドカーで理解する L7 トラフィック制御](./infra/envoy_ja.md) — **中級**
  - Envoy プロキシを手動配置してトラフィック制御（リトライ・タイムアウト・ルーティング）。推奨: 先に [Traefik 実習](./infra/traefik_ja.md)（サイドカーによる内部トラフィック制御の前にエッジルーティングを学ぶ）
- [OAuth2 実習：Keycloak + Go + Podman で学ぶ認可フロー](./infra/oauth2_ja.md) — **中級**
  - Keycloak でトークン発行し、Go Resource Server で JWT を検証して保護 API を実装
- [Webセキュリティ実習（全 2 パート）：XSS, CSRF, クリックジャッキングを体験する](./infra/web_security_ja.md) — **中級**
  - 意図的に脆弱なアプリで代表的な Web 攻撃を体験。パート 2（[防御実践編](./infra/web_security_defense_ja.md)）の前に必ずパート 1 を完了してください。パート 2 では victim アプリのコピーに各対策を適用し、攻撃が阻止されることを再検証します
- [サプライチェーンセキュリティ実習：依存ライブラリとコンテナイメージの脆弱性管理](./infra/supply_chain_security_ja.md) — **中級**
  - govulncheck/Trivy で脆弱性検出、syft/grype で SBOM 生成、cosign でイメージ署名・検証を体験。推奨: 攻撃者視点の理解として先に [Webセキュリティ実習](./infra/web_security_ja.md)
- [Terraform 実習：基礎から実践的な DNS サーバー構築まで](./infra/terraform_ja.md) — **中級**
  - local プロバイダーで基礎を学び、Docker/Podman で CoreDNS 環境を IaC 管理

#### 上級

- [コンテナランタイム実習：namespaces と cgroups で理解するコンテナの正体](./infra/container_runtime_ja.md) — **上級**
  - 手動で namespace/cgroup を作成し、`podman run` の裏側を再現
- [TCP/IP プロトコルスタック実習（全 2 パート）：スクラッチからネットワークスタックを構築する](./infra/tcpip_stack_ja.md) — **上級**
  - Go と C (CGO) で Ethernet/IP/ICMP を実装し、raw socket からパケットを理解。パート 2（[UDP・スタック統合](./infra/tcpip_stack_udp_ja.md)）の前にパート 1 を完了してください
- [K8s Service (LoadBalancer) 実習：iptables で作る仮想ロードバランサ](./infra/k8s_lb_ja.md) — **上級**
  - VIP 追加 + DNAT/SNAT で MetalLB・kube-proxy の動作原理を再現。前提: [Podman VLAN 実習](./infra/vlan_ja.md)（podman ネットワーク・ルータコンテナ・VLAN 10/20 の構成を再利用）
- [Zero Trust Architecture 実習：mTLS と Service Mesh による「常に検証」の実現](./infra/zero_trust_ja.md) — **上級**
  - Istio と SPIRE を活用し、ネットワーク境界に頼らない身元ベースのセキュリティ（mTLS/認可）を実装。推奨: 先に [Kubernetes 実習](./infra/kubernetes_basics_ja.md) と [TLS 実習](./infra/tls_ja.md)
- [iSCSI と Kubernetes PersistentVolume 実習](./infra/iscsi_pv_ja.md) — **上級**
  - iSCSI ターゲット/イニシエータを構築し、K8s PV/PVC でマウント・永続化を確認。前提: [Kubernetes 実習](./infra/kubernetes_basics_ja.md)（STEP 6 から PV/PVC の概念を使用）
- [Kubernetes 運用・セキュリティ実習：本番運用のための高度な機能](./infra/kubernetes_operations_ja.md) — **上級**
  - RBAC, NetworkPolicy, HPA, GitOps (ArgoCD), Backup (Velero) 等、プロダクション運用に必要な技術を体験。前提: [Kubernetes 実習](./infra/kubernetes_basics_ja.md) を先に完了
- [Kerberos / SPNEGO 認証実習：NGINX と Podman でシングルサインオンを体験する](./infra/kerberos_ja.md) — **上級**
  - KDC 構築 + NGINX SPNEGO モジュールで、パスワードレスな認証フローを体験
- [OAuth2 実習：Microsoft Entra ID + Go で学ぶ認可フロー](./infra/oauth2_entra_ja.md) — **上級**
  - Managed Identity を使う M2M と、ユーザー委任での API アクセスを Microsoft Entra ID で学習。推奨: 同じフローをローカル Keycloak で扱う [OAuth2 (Keycloak) 実習](./infra/oauth2_ja.md) を先に
- [シャーディング基礎: Sparse Table, Segment Tree, Elasticsearch, Cortex](./infra/sharding_ja.md) — **上級**
  - Elasticsearch/Cortex を題材にしたシャーディング設計の実践ガイド（Cortex PR #7266 / #7270 の要点を含む、日本語）。推奨: [Redis 実習](./infra/cache_ja.md)（ストレージ層のスケーリングパターン）

#### リファレンス

- [インフラ勉強会用語集](./infra/glossary_ja.md)
  - 各ワークショップで登場する専門用語の簡潔な説明（ネットワーク、コンテナ、K8s、セキュリティ等）

#### インフラ: 今後追加予定のコンテンツ

- 監視基盤実習：Prometheus + Grafana によるメトリクス収集と可視化
  - Node Exporter + Prometheus + Grafana でメトリクス収集・ダッシュボード構築

### ソフトウェア・アーキテクチャ関連

#### 初級

- [クリーンアーキテクチャ実習 (WS8): 最小構成で本質を理解する](./software/clean_arch_ws8_ja.md) — **初級**
  - 小さな挨拶アプリで依存ルールを追い、ビジネスルールとインフラの差し替えを実践。**ここから開始**: クリーンアーキテクチャの予備知識は不要。WS8 の後、[クリーンアーキテクチャ解説](./software/clean_arch_ja.md) を読んで WS1 へ進んでください
- [クリーンアーキテクチャ（3層バリアント: Adapters / UseCases / Domain）](./software/clean_arch_ja.md)
  - Domain/UseCase/Adapters の依存関係と責務を図解で解説。WS8 と WS1 の間に読むのがおすすめです
- [クリーンアーキテクチャ実習 (WS1): 変更に強い設計を学ぶ](./software/clean_arch_ws1_ja.md) — **初級**
  - Entity・Domain Service の実装と、DB から AD へのインフラ差し替えを体験。推奨: 先に WS8 と上記の解説
- [SOLID principles (Goで学ぶ設計原則)](./software/solid_ja.md) — **初級**
  - S/O/L/I/D を Go の実例と Clean Architecture の対応で理解

#### 中級

- [クリーンアーキテクチャ実習 (WS2): 拡張と最適化](./software/clean_arch_ws2_ja.md) — **中級**
  - 通知チャネルの抽象化と Decorator パターンによる透過的キャッシュ導入。前提: [WS1](./software/clean_arch_ws1_ja.md)
- [クリーンアーキテクチャ実習 (WS3): 通信プロトコルの差し替え](./software/clean_arch_ws3_ja.md) — **中級**
  - BBS の REST API を gRPC に移行し、Presentation Adapter と Composition Root だけの変更で済むことを体験。前提: [WS1](./software/clean_arch_ws1_ja.md) と [WS2](./software/clean_arch_ws2_ja.md)（または Port と Composition Root に慣れていること）
- [クリーンアーキテクチャ実習 (WS4): ビジネスルールの追加](./software/clean_arch_ws4_ja.md) — **中級**
  - 新しいビジネスルールを追加し、内側から外側への変更波及を体験。前提: [WS1](./software/clean_arch_ws1_ja.md)・[WS2](./software/clean_arch_ws2_ja.md) の理解。WS3 は任意の事前学習
- [クリーンアーキテクチャ実習 (WS5): 永続化層の差し替え](./software/clean_arch_ws5_ja.md) — **中級**
  - SQLite から PostgreSQL への移行を Infra Adapter と Composition Root の変更で実現。前提: [WS4](./software/clean_arch_ws4_ja.md) — 生成ツールで WS4 完了相当のコピーから開始します
- [クリーンアーキテクチャ実習 (WS6): 外部サービス統合](./software/clean_arch_ws6_ja.md) — **中級**
  - Slack 通知を新しい Port/Adapter パターンで追加。前提: [WS5](./software/clean_arch_ws5_ja.md) — 生成ツールで WS5 完了相当のコピーから開始します
- [クリーンアーキテクチャ実習 (WS7): 認証の追加](./software/clean_arch_ws7_ja.md) — **中級**
  - JWT Bearer Token 認証を Presentation Adapter の横断的関心として追加。前提: [WS1](./software/clean_arch_ws1_ja.md)〜[WS4](./software/clean_arch_ws4_ja.md)（または BBS プロジェクトのビルドとテストが通ること）
- [冪等性パターン：安全な再試行を実現する設計](./software/idempotency_ja.md) — **中級**
  - Idempotency Key で重複処理を防ぎ、並行実行、有効期限、障害処理のトレードオフを学習。[RabbitMQ 実習](./infra/rabbitmq_ja.md)（リトライ安全なコンシューマ）と組み合わせると効果的
- [Go キャッシュパターン入門](./software/go_cache_patterns_ja.md) — **中級**
  - Go の代表的なキャッシュパターンを整理（TTL/LRU/Cache-Aside/Redis/Write-Through）。リファレンスカタログ — 必要なパターンから直接読めます
- [Go 言語におけるメッセージキューパターン](./software/go_mq_patterns_ja.md) — **中級**
  - チャネル、Fan-out、Worker Pool、Kafka、Retry/DLQ を実行可能な Go の例で比較。リファレンスカタログ — 必要なパターンから直接読めます
- [セキュアコーディング Best Practices (Go)](./software/secure_coding_ja.md) — **中級**
  - SQL インジェクション、パストラバーサル、パスワードハッシュ、競合状態、情報漏洩防止など日常のコーディングで気をつけるべき作法を解説

#### 上級

- [Succinct データ構造入門（Go視点）](./software/succinct_ja.md) — **上級**
  - LOUDS を用いた省メモリな Trie 実装の解説

#### リファレンス

- [クリーンアーキテクチャ参考図（簡略版／英語）](./software/clean_arch.md)
  - アーキテクチャ図と主要原則を凝縮したクイックリファレンス（英語）
- [Design Patterns](https://github.com/sokoide/design-patterns/README_ja.md)
  - GoF デザインパターンの実装例と適用場面

## メンテナンス

`pnpm install` または `npm install` で Markdown ツールを導入します（Python 3.9 以上も必要です）。
`make format` は同じファイル一覧に Prettier、markdownlint、textlint を適用します。
`CLAUDE.md`、依存関係のキャッシュ、ツールのメタデータは除外し、ツールが失敗すると処理を停止します。
`make test-tools` で保守スクリプトをテストし、`make check-headings` で日英ペアの見出しレベルの順序を比較します。
後者は不一致や読み取りエラーで非ゼロ終了します。翻訳の正確さや、対となるファイルの欠落は検証しません。

`make check-go` で全 Go モジュールのビルド、静的検査、race 検出付きテストを実行します。ログとスキップの詳細は `.cache/go-validation/` に保存します。`make check-go GO_CHECK_FLAGS=--require-integration` はテストのスキップや Linux 専用部分の未検証も失敗として扱います。Raw Socket の実行には Linux と CGO、RabbitMQ と Vault の統合テストには各サービスが必要です。

`make check-docs` は Go 要件とアセットの整合も確認します。`make check-bbs-states` は BBS の開始・完成状態を検証し、`make check` にも含まれます。
