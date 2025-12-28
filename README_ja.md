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
- TLS/SSL 証明書実習：自己署名 CA 構築と証明書チェーンを理解する（予定）
  - OpenSSL で CA 構築 → Traefik/CoreDNS で HTTPS 化
- リバースプロキシ実習：Traefik で学ぶ K8s Ingress の裏側（予定）
  - Traefik + 複数バックエンドコンテナで SSL 終端・ルーティング・ヘルスチェック
- サービスメッシュ基礎実習：Envoy サイドカーで理解する L7 トラフィック制御（予定）
  - Envoy プロキシを手動配置してトラフィック制御（リトライ・タイムアウト・ルーティング）
- コンテナランタイム実習：namespaces と cgroups で理解するコンテナの正体（予定）
  - 手動で namespace/cgroup を作成し、`docker run` の裏側を再現
- メッセージキュー実習：RabbitMQ で学ぶ Pub/Sub とキューイング（予定）
  - RabbitMQ + Producer/Consumer コンテナで非同期メッセージング
- オブジェクトストレージ実習：MinIO で学ぶ S3 互換 API（予定）
  - MinIO 構築 + AWS CLI/SDK でバケット操作・署名付き URL 生成
- 監視基盤実習：Prometheus + Grafana によるメトリクス収集と可視化（予定）
  - Node Exporter + Prometheus + Grafana でメトリクス収集・ダッシュボード構築
- シークレット管理実習：HashiCorp Vault による動的シークレット管理（予定）
  - Vault 構築 + アプリからの動的シークレット取得

### ソフトウェア・アーキテクチャ関連

- [クリーンアーキテクチャ (3 レイヤー構成)](./software/clean_arch_ja.md)
  - Domain/UseCase/Infra の依存関係と責務を図解で解説
- [クリーンアーキテクチャ実習 (WS1): 変更に強い設計を学ぶ](./software/clean_arch_ws1_ja.md)
  - Entity・Domain Service の実装と、DB から AD へのインフラ差し替えを体験
- [クリーンアーキテクチャ実習 (WS2): 拡張と最適化](./software/clean_arch_ws2_ja.md)
  - 通知チャネルの抽象化と Decorator パターンによる透過的キャッシュ導入
- [クリーンアーキテクチャ実習 (WS3): e-commerce platform](./software/clean_arch_ws3_ja.md)
  - 注文・在庫管理システムを題材に、マイクロサービス分離を見据えた設計
- [Design Patterns](https://github.com/sokoide/design-patterns/README_ja.md)
  - GoF デザインパターンの実装例と適用場面
