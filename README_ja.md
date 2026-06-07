# ワークショップリポジトリ

---

🌐 Available languages:
[English](./README.md) | [日本語](./README_ja.md)

---

このリポジトリは、さまざまな技術トピックに関する実習用の資料を保管します。

## 内容

### インフラ関連

- [CoreDNS 実習：親子 DNS サーバーを構築して名前解決を理解する](./infra/coredns_ja.md)
  - 権威サーバーとフォワーディングを 2 台の VM で構築し、ゾーン委譲の仕組みを体験
- [Podman 実習：macvlan + ルータコンテナで L3 分離ネットワークを作る](./infra/vlan_ja.md)
  - VLAN サブインタフェース上に macvlan ネットワークを作成し、ルータコンテナで NAT/転送
- [K8s Service (LoadBalancer) 実習：iptables で作る仮想ロードバランサ](./infra/k8s_lb_ja.md)
  - VIP 追加 + DNAT/SNAT で MetalLB・kube-proxy の動作原理を再現
- [iSCSI と Kubernetes PersistentVolume 実習](./infra/iscsi_pv_ja.md)
  - iSCSI ターゲット/イニシエータを構築し、K8s PV/PVC でマウント・永続化を確認
- [Terraform 実習：基礎から実践的な DNS サーバー構築まで](./infra/terraform_ja.md)
  - local プロバイダーで基礎を学び、Docker/Podman で CoreDNS 環境を IaC 管理
- [TLS/SSL 証明書実習：自己署名 CA 構築と証明書チェーンを理解する](./infra/tls_ja.md)
  - OpenSSL で CA 構築 → 証明書の検証までの一連の流れを体験
- [コンテナランタイム実習：namespaces と cgroups で理解するコンテナの正体](./infra/container_runtime_ja.md)
  - 手動で namespace/cgroup を作成し、`podman run` の裏側を再現
- [Redis 実習：Sorted Sets で作るリアルタイム・ゲームランキング](./infra/cache_ja.md)
  - Redis の Sorted Sets を活用した高速なランキングシステム構築と、Sets による不正ユーザー管理を学習
- [メッセージキュー実習：RabbitMQ で学ぶ Pub/Sub とキューイング](./infra/rabbitmq_ja.md)
  - RabbitMQ の Topic Exchange を活用したリアルタイム仮想通貨監視システム構築
- [シークレット管理実習：HashiCorp Vault による安全な API キー管理](./infra/secret_management_ja.md)
  - Vault KV v2 を活用したシークレットの集中管理・バージョニング・Clean Architecture による統合
- [Kerberos / SPNEGO 認証実習：NGINX と Podman でシングルサインオンを体験する](./infra/kerberos_ja.md)
  - KDC 構築 + NGINX SPNEGO モジュールで、パスワードレスな認証フローを体験
- [OAuth2 実習：Keycloak + Go + Podman で学ぶ認可フロー](./infra/oauth2_ja.md)
  - Keycloak でトークン発行し、Go Resource Server で JWT を検証して保護 API を実装
- [OAuth2 実習：Microsoft Entra ID + Go で学ぶ認可フロー](./infra/oauth2_entra_ja.md)
  - Managed Identity を使う M2M と、ユーザー委任での API アクセスを Microsoft Entra ID で学習
- [リバースプロキシ実習：Traefik で学ぶ K8s Ingress の裏側](./infra/traefik_ja.md)
  - Traefik + 複数バックエンドコンテナで SSL 終端・ルーティング・ヘルスチェック
- [サービスメッシュ基礎実習：Envoy サイドカーで理解する L7 トラフィック制御](./infra/envoy_ja.md)
  - Envoy プロキシを手動配置してトラフィック制御（リトライ・タイムアウト・ルーティング）
- [オブジェクトストレージ実習：MinIO で学ぶ S3 互換 API](./infra/minio_ja.md)
  - MinIO 構築 + AWS CLI/SDK でバケット操作・署名付き URL 生成
- [HTTP 永続接続実習：Keep-Alive, WebSocket, HTTP/2, gRPC, HTTP/3](./infra/http_persistent_ja.md)
  - HTTP/1.1 から最新の HTTP/3 (QUIC) まで、接続再利用と 0-RTT の仕組みを比較
- [シャーディング基礎: Sparse Table, Segment Tree, Elasticsearch, Cortex](./infra/sharding_ja.md)
  - Elasticsearch/Cortex を題材にしたシャーディング設計の実践ガイド（Cortex PR #7266 / #7270 の要点を含む、日本語）。
- [TCP/IP プロトコルスタック実習：スクラッチからネットワークスタックを構築する](./infra/tcpip_stack_ja.md)
  - Go と C (CGO) で Ethernet/IP/ICMP/UDP を実装し、raw socket からパケットを理解
- [Kubernetes 実習：コンテナオーケストレーションの基礎から実践まで](./infra/kubernetes_basics_ja.md)
  - Pod, Deployment, Service, ConfigMap, Helm 等、K8s の基本リソースとエコシステムを網羅的に学習
- [Kubernetes 運用・セキュリティ実習：本番運用のための高度な機能](./infra/kubernetes_operations_ja.md)
  - RBAC, NetworkPolicy, HPA, GitOps (ArgoCD), Backup (Velero) 等、プロダクション運用に必要な技術を体験
- [Zero Trust Architecture 実習：mTLS と Service Mesh による「常に検証」の実現](./infra/zero_trust_ja.md)
  - Istio と SPIRE を活用し、ネットワーク境界に頼らない身元ベースのセキュリティ（mTLS/認可）を実装
- [Webセキュリティ実習：XSS, CSRF, クリックジャッキングを体験する](./infra/web_security_ja.md)
  - 意図的に脆弱なアプリを用い、代表的な Web 攻撃の仕組みと対策（エスケープ、トークン等）を学習
- [サプライチェーンセキュリティ実習：依存ライブラリとコンテナイメージの脆弱性管理](./infra/supply_chain_security_ja.md)
  - govulncheck/Trivy で脆弱性検出、syft/grype で SBOM 生成、cosign でイメージ署名・検証を体験

#### インフラ: 今後追加予定のコンテンツ

- 監視基盤実習：Prometheus + Grafana によるメトリクス収集と可視化
  - Node Exporter + Prometheus + Grafana でメトリクス収集・ダッシュボード構築

### ソフトウェア・アーキテクチャ関連

- [クリーンアーキテクチャ（3層バリアント: Adapters / UseCases / Domain）](./software/clean_arch_ja.md)
  - Domain/UseCase/Adapters の依存関係と責務を図解で解説
- [クリーンアーキテクチャ実習 (WS1): 変更に強い設計を学ぶ](./software/clean_arch_ws1_ja.md)
  - Entity・Domain Service の実装と、DB から AD へのインフラ差し替えを体験
- [クリーンアーキテクチャ実習 (WS2): 拡張と最適化](./software/clean_arch_ws2_ja.md)
  - 通知チャネルの抽象化と Decorator パターンによる透過的キャッシュ導入
- [クリーンアーキテクチャ実習 (WS3): 通信プロトコルの差し替え](./software/clean_arch_ws3_ja.md)
  - BBS の REST API を gRPC に移行し、Framework 層だけの変更で済むことを体験
- [クリーンアーキテクチャ実習 (WS4): ビジネスルールの追加](./software/clean_arch_ws4_ja.md)
  - 新しいビジネスルールを追加し、内側から外側への変更波及を体験
- [クリーンアーキテクチャ実習 (WS5): 永続化層の差し替え](./software/clean_arch_ws5_ja.md)
  - SQLite から PostgreSQL への移行を Infra 層だけの変更で実現
- [クリーンアーキテクチャ実習 (WS6): 外部サービス統合](./software/clean_arch_ws6_ja.md)
  - Slack 通知を新しい Port/Adapter パターンで追加
- [クリーンアーキテクチャ実習 (WS7): 認証の追加](./software/clean_arch_ws7_ja.md)
  - JWT Bearer Token 認証を Framework 層の横断的関心として追加
- [SOLID principles (Goで学ぶ設計原則)](./software/solid_ja.md)
  - S/O/L/I/D を Go の実例と Clean Architecture の対応で理解
- [セキュアコーディング Best Practices (Go)](./software/secure_coding_ja.md)
  - SQL インジェクション、パストラバーサル、パスワードハッシュ、競合状態、情報漏洩防止など日常のコーディングで気をつけるべき作法を解説
- [Go キャッシュパターン入門](./software/go_cache_patterns_ja.md)
  - Go の代表的なキャッシュパターンを整理（TTL/LRU/Cache-Aside/Redis/Write-Through、日本語）
- [Succinct データ構造入門（Go視点）](./software/succinct_ja.md)
  - LOUDS を用いた省メモリな Trie 実装の解説
- [Design Patterns](https://github.com/sokoide/design-patterns/README_ja.md)
  - GoF デザインパターンの実装例と適用場面
