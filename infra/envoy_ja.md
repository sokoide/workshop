# サービスメッシュ基礎実習：Envoy サイドカーで理解する L7 トラフィック制御

この実習では、モダンなサービスメッシュのデータプレーンとして事実上の標準となっている **Envoy Proxy** を使用し、アプリケーションコードを変更することなく、高度なトラフィック制御（リトライ、タイムアウト、サーキットブレーカー）を実現する方法を学びます。

## ゴール

この実習を完了すると、以下のことができるようになります：

- サイドカーパターン（アプリケーションの横にプロキシを配置する構成）を理解する
- Envoy の設定ファイル（リスナー、クラスター、ルート）の構造を理解する
- ネットワーク障害をシミュレートし、リトライやタイムアウトによる回復力を確認する
- Envoy Admin インターフェースを使用して統計情報を確認する

---

## 課題と解決策

マイクロサービスアーキテクチャでは、ネットワーク通信が不安定になることが避けられません。

### ❌ 従来の課題（Resiliency Logic in Code）

- 各マイクロサービスのコード内にリトライやタイムアウトの実装が必要
- 言語ごとにライブラリが異なり、挙動が統一できない
- 設定変更のためにアプリケーションの再デプロイが必要

### ✅ サービスメッシュ（Envoy）のアプローチ

- **Out-of-Process**: 通信制御ロジックをアプリケーションから切り離し、サイドカープロキシに移譲
- **言語非依存**: アプリが Go, Python, Java 何であっても同じ制御が可能
- **可観測性**: すべての通信メトリクスを統一的に収集

---

## アーキテクチャ

クライアントからのリクエストを Envoy が受け取り、バックエンドサービスへ転送します。
本実習では、バックエンドアプリに「意図的に遅延する」「エラーを返す」機能を実装し、それに対して Envoy がどう対処するかを確認します。

```mermaid
graph LR
    Client[Client] -- HTTP --> Envoy[Envoy Proxy (Sidecar)]
    
    subgraph Service Mesh Pod
        Envoy -- Localhost --> App[Backend Service]
    end

    Envoy -.->|Stats/Admin| Admin[Admin Interface :9901]
```

### 想定ディレクトリ構造

```text
infra/assets/envoy/
├── envoy.yaml              # Envoy 設定ファイル
├── docker-compose.yml      # 構成定義
└── app/                    # テスト用バックエンドアプリ (遅延・エラー生成機能付き)
```

---

## 準備

### 1. Envoy 設定ファイルの作成 (`envoy.yaml`)

以下の設定では、バックエンドへの接続に対し **2秒のタイムアウト** と **3回のリトライ** を定義しています。

```yaml
static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address: { address: 0.0.0.0, port_value: 10000 }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          route_config:
            name: local_route
            virtual_hosts:
            - name: local_service
              domains: ["*"]
              routes:
              - match: { prefix: "/" }
                route: 
                  cluster: service_backend
                  timeout: 2s
                  retry_policy:
                    retry_on: "5xx"
                    num_retries: 3
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router

  clusters:
  - name: service_backend
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: service_backend
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address: { address: backend, port_value: 80 }
```

### 2. 環境の起動

```bash
cd infra/assets/envoy
docker-compose up -d
```

---

## 実習ステップ

### STEP 1: プロキシ経由のアクセス

Envoy（ポート 10000）経由でバックエンドにアクセスできることを確認します。

```bash
curl -v http://localhost:10000/
```

レスポンスヘッダーに `server: envoy` が含まれていることを確認してください。これは Envoy を経由した証拠です。

### STEP 2: タイムアウトの動作確認

バックエンドサービスが遅延した場合の Envoy の挙動を確認します。
`timeout: 2s` と設定されているため、バックエンドに **3秒のスリープ** を要求します（実習用アプリの機能）。

```bash
curl -v http://localhost:10000/sleep/3
```

**結果**: Envoy が 2 秒経過した時点で接続を切り、`504 Gateway Timeout` を返すことを確認します。これにより、バックエンドの不調が原因でクライアントが無限に待たされるのを防ぎます。

### STEP 3: リトライの動作確認

設定ファイルには `num_retries: 3` が設定されています。バックエンドが `500` エラーを返すようにリクエストします。

```bash
curl -v http://localhost:10000/error/500
```

Envoy のログまたは統計情報（Admin IF）を確認すると、実際にはバックエンドに対して複数回のリクエストが送信されたことがわかります。一時的なネットワークエラーであれば、ユーザーにはエラーを見せずに成功させることができます。

### STEP 4: Admin インターフェース

Envoy の内部状態を確認します。

`http://localhost:9901/stats`

ここで `retry` や `timeout` の発生回数がメトリクスとして記録されていることを確認します。これはシステムの健全性を監視するために重要です。

---

## Clean Architecture との関連

Clean Architecture における「Framework & Drivers」層（詳細）として Envoy を捉えることができます。
ビジネスロジック（Usecase/Domain）は、ネットワークの不安定さやリトライ制御といった「インフラの詳細」から守られ、純粋なロジックに集中できます。インフラストラクチャの責務をコードからサイドカーへ委譲する究極の形です。

---

## 片付け

```bash
docker-compose down
```

---

## 次のステップ

- **Circuit Breaking**: 大量のエラー発生時にシステム全体を守る遮断機能
- **Istio**: Envoy をコントロールプレーンで集中管理するサービスメッシュ製品へのステップアップ
