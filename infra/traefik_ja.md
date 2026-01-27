# リバースプロキシ実習：Traefik で学ぶ K8s Ingress の裏側

この実習では、クラウドネイティブなエッジルーターである **Traefik** を使用して、Kubernetes Ingress コントローラーの背後にある仕組み（L7 ルーティング、ロードバランシング、サービスディスカバリ）を学びます。

## ゴール

この実習を完了すると、以下のことができるようになります：

- L7 リバースプロキシの基本機能（パスベース/ホストベースのルーティング）を理解する
- 複数のバックエンドサービスに対するロードバランシングを設定する
- ミドルウェアを使用したリクエストの加工（Basic 認証、ヘッダー追加）を行う
- コンテナの動的な追加・削除を検知するサービスディスカバリの仕組みを理解する

---

## 従来の課題（静的プロキシの限界）

Nginx などの従来のプロキシでは、バックエンドサーバーが増減するたびに設定ファイル（`nginx.conf`）を書き換えてリロードする必要がありました。

### ❌ 課題

- コンテナの IP アドレスが変わるたびに設定変更が必要
- 設定変更時のダウンタイムやオペレーションミスのリスク
- マイクロサービス環境での管理コスト増大

### ✅ Traefik のアプローチ

- **サービスディスカバリ**: Docker や Kubernetes API を監視し、設定を自動更新
- **動的設定**: 再起動なしでルーティングルールを即座に反映
- **ミドルウェア**: 認証やレート制限などをプラグインのように適用可能

---

## アーキテクチャ

Docker プロバイダーを使用して、コンテナラベルに基づいた動的なルーティングを構築します。

```mermaid
graph LR
    Client[Client (curl/Browser)]
    Traefik[Traefik Proxy]
    
    subgraph Backends [Backend Containers]
        App1[Whoami App A]
        App2[Whoami App B]
        App3[Nginx App]
    end

    Client -- "Host: app.local" --> Traefik
    Client -- "Path: /api" --> Traefik
    
    Traefik -- "Load Balancing" --> App1 & App2
    Traefik -- "Routing" --> App3

    Traefik -.->|Watch Events| DockerSocket((Docker Socket))
```

### 想定ディレクトリ構造

```text
infra/assets/traefik/
├── docker-compose.yml      # Traefik とバックエンドの定義
└── Makefile                # 操作コマンド
```

---

## 準備

### 1. Docker Compose 定義の作成

以下の内容で `docker-compose.yml` を作成します（実習用アセットとして提供される想定）。

```yaml
version: "3"

services:
  # Traefik リバースプロキシ
  traefik:
    image: traefik:v2.10
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
    ports:
      - "80:80"
      - "8080:8080" # Dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro

  # バックエンド A (Whoami)
  whoami-a:
    image: traefik/whoami
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.whoami.rule=Host(`whoami.localhost`)"
      - "traefik.http.services.whoami.loadbalancer.server.port=80"

  # バックエンド B (Whoami - 同一サービスとしてLB)
  whoami-b:
    image: traefik/whoami
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.whoami.rule=Host(`whoami.localhost`)"
      - "traefik.http.services.whoami.loadbalancer.server.port=80"
```

### 2. コンテナの起動

```bash
cd infra/assets/traefik
docker-compose up -d
```

---

## 実習ステップ

### STEP 1: ダッシュボードの確認

ブラウザで `http://localhost:8080` にアクセスします。
Traefik が Docker 上で実行されているサービス（`whoami-a`, `whoami-b`）を自動的に検出していることを確認します。

### STEP 2: ホストベース・ルーティングとロードバランシング

`whoami.localhost` へのリクエストを送信し、Traefik が 2 つのコンテナにリクエストを振り分けていることを確認します。

```bash
# 1回目
curl -H "Host: whoami.localhost" http://localhost
# Output: Hostname: <Container-ID-A>

# 2回目
curl -H "Host: whoami.localhost" http://localhost
# Output: Hostname: <Container-ID-B>
```

**ポイント**: `docker-compose.yml` で同じ `Host` ルールを持つコンテナを複数起動するだけで、Traefik が自動的にロードバランシングを行います。

### STEP 3: パスベース・ルーティングの追加

新しい Nginx コンテナを追加し、`/nginx` パスでアクセスできるようにします。これは `docker-compose.yml` を編集せずに、`docker run` で動的に追加できます。

```bash
docker run -d --name nginx-demo \
  --label "traefik.enable=true" \
  --label "traefik.http.routers.nginx.rule=Host(`localhost`) && PathPrefix(`/nginx`)" \
  --label "traefik.http.services.nginx.loadbalancer.server.port=80" \
  nginx:alpine
```

確認：

```bash
curl http://localhost/nginx
```

### STEP 4: ミドルウェアの適用（Basic 認証）

Traefik の強力な機能であるミドルウェアを試します。特定のルートに Basic 認証をかけます。

```bash
# パスワード生成 (user:password)
htpasswd -nb user password
# user:$apr1$Sr...
```

ラベルを追加してコンテナを再起動することで、アプリケーションコードを変更せずに認証層を追加できます。

---

## 技術的なポイント

### Ingress との関係

Kubernetes の Ingress リソースは、この実習で設定した「ルーティングルール」の標準化された定義です。Traefik は Ingress Controller として動作し、K8s API から Ingress リソースを読み取って、同様のルーティングを実行します。

### サービスメッシュへの入り口

Traefik は主に「エッジ（外部から内部）」のトラフィックを扱いますが、内部のサービス間通信（East-West）を制御するのが次のステップである「サービスメッシュ（Envoy）」です。

---

## 片付け

```bash
docker-compose down
docker rm -f nginx-demo
```

---

## 次のステップ

- **HTTPS 化**: Let's Encrypt と連携した自動証明書発行
- **Kubernetes へのデプロイ**: Helm チャートを使用した Traefik の導入
- **サービスメッシュ**: Envoy を使用したより高度なトラフィック制御（次の実習）
