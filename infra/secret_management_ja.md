# シークレット管理実習：HashiCorp Vault による安全な API キー管理

この実習では、**HashiCorp Vault** と **Go** を使用して、API キー、データベース認証情報、その他のセンシティブデータをソースコードにハードコードすることなく安全に管理するシステムを構築します。

## ゴール

この実習を完了すると、以下のことができるようになります：

- シークレット管理がなぜ重要かを理解する
- 一般的なアンチパターン（ハードコード、.env ファイル）を識別・回避する
- HashiCorp Vault の基本を習得（KV v2 シークレットエンジン、トークン、リース）
- Clean Architecture を使用して Vault をアプリケーションに統合する
- シークレットのバージョニングとローテーションを実装する

---

## 避けるべきアンチパターン

正しい方法を学ぶ前に、よくある間違いを見てみましょう：

### ❌ アンチパターン 1：ハードコードされた認証情報

これが**最悪**の手法です。認証情報は Git 履歴に永遠に残ります：

```go
// infra/assets/rabbitmq_crypto/cmd/ticker/main.go より
url := "amqp://guest:guest@localhost:5672/"  // 決してこうしてはいけない
```

**問題点：**

- Git 履歴にシークレットが永遠に残る（後から削除しても）
- リポジトリへのアクセス権を持つ誰もが認証情報を見れる
- コードレビューやプルリクエストでシークレットが露出する
- 認証情報をローテーションするにはコードを変更する必要がある

### ❌ アンチパターン 2：環境変数ファイル (.env)

ハードコードよりはマシですが、`.env` ファイルにはまだリスクがあります：

```bash
# .env ファイル（git に誤ってコミットされがち）
DATABASE_URL=postgresql://user:password123@localhost/db
API_KEY=sk-live-abc123xyz789
```

**問題点：**

- `.env` を Git に誤ってコミットしやすい
- 誰がシークレットにアクセスしたかの監査履歴がない
- 自動ローテーションがない
- ディスク上に平文で保存される

### ✅ 正しい方法：外部シークレットマネージャー

Vault は以下を提供します：

- シークレットの中央集権管理
- 監査ログ
- 自動シークレットローテーション
- きめ細かなアクセス制御
- 保存時の暗号化

---

## アーキテクチャ

この実習では、適切な関心の分離を持つ **Clean Architecture** を実装します：

```mermaid
graph TB
    subgraph Framework ["Framework Layer (cmd/)"]
        GS[get-secret]
        PS[put-secret]
        LS[list-secrets]
        AC[api-client]
    end

    subgraph Usecase ["Usecase Layer (usecase/)"]
        SM[SecretManager]
        API[APIClient]
    end

    subgraph Domain ["Domain Layer (domain/)"]
        Entity[Secret エンティティ]
        Repo[SecretRepository インターフェース]
    end

    subgraph Infra ["Infra Layer (infra/vault/)"]
        VA[Vault アダプター]
        SDK[Vault Go SDK]
    end

    subgraph External ["External"]
        V[Vault Server KV v2]
    end

    GS --> SM
    PS --> SM
    LS --> SM
    AC --> API
    SM --> Repo
    API --> SM
    VA -.->|実装| Repo
    VA --> SDK
    SDK --> V

    style Repo fill:#e1f5fe
    style VA fill:#fff3e0
    style V fill:#f3e5f5
```

### レイヤー構造とディレクトリ

```text
infra/assets/secret_management/
├── cmd/                        # Framework Layer (エントリーポイント)
│   ├── get-secret/main.go      # シークレット取得
│   ├── put-secret/main.go      # シークレット格納
│   ├── list-secrets/main.go    # シークレット一覧
│   └── api-client/main.go      # Vault 認証付き API クライアント
├── domain/                     # Domain Layer (純粋な Go)
│   ├── secret.go               # Secret エンティティ
│   └── repository.go           # SecretRepository インターフェース
├── usecase/                    # Usecase Layer (ビジネスロジック)
│   ├── secret_manager.go       # シークレット管理のオーケストレーション
│   └── api_client.go           # API クライアントパターン
├── infra/                      # Infra Layer (Vault アダプター)
│   └── vault/
│       ├── client.go           # Vault クライアント設定
│       ├── secret.go           # KV v2 実装
│       └── secret_test.go      # 統合テスト
├── Makefile                    # コンテナ管理
└── go.mod
```

### 重要な原則

- **Domain Layer**: 外部依存のない純粋な Go。`SecretRepository` インターフェースを定義。
- **Usecase Layer**: ドメインインターフェースのみに依存するビジネスロジック。
- **Infra Layer**: HashiCorp Vault SDK を使用して `SecretRepository` を実装する Vault アダプター。
- **Framework Layer**: 依存関係をまとめる CLI コマンド。

---

## 準備

### 1. Vault の起動

Vault Dev コンテナを起動します。Dev モードは**安全ではありません**が、学習には最適です：

```bash
cd infra/assets/secret_management
make vault-up
```

これにより以下の設定で Vault が起動します：

- アドレス: `http://localhost:8200`
- ルートトークン: `dev-workshop-token`
- 自動アンシール（Dev モードのみ）

### 2. KV v2 シークレットエンジンの有効化

シークレットを保存する前に、KV v2 シークレットエンジンを有効にする必要があります。
ローカルに `vault` コマンドがない場合でも、`make init` を使用してコンテナ内で設定を行えます：

```bash
# KV v2 を有効化（コンテナ内でコマンドを実行）
make init
```

または、手動でコンテナ内のコマンドを実行することもできます：

```bash
podman exec workshop-vault vault secrets enable -path=secret kv-v2
```

**注:** KV v2 はすべてのシークレットのバージョニングを提供し、変更の追跡と必要に応じたロールバックを可能にします。

### 3. 依存ライブラリのインストール

```bash
go mod tidy
```

---

## 実習ステップ

### STEP 1: アンチパターンの確認 (5分)

Vault を使用する前に、シークレットが誤って扱われがちな例を見てみましょう：

```bash
cd ../rabbitmq_crypto
grep -r "guest:guest" cmd/
# 結果: url := "amqp://guest:guest@localhost:5672/"
```

**重要なポイント:** 一度シークレットが Git にコミットされると、（履歴からも）永遠に残ります。

### STEP 2: 基本的なシークレット操作 (15分)

シークレット管理プロジェクトに戻ります：

```bash
cd ../secret_management
```

#### ステップ 1: 最初のシークレットを保存する

`put-secret` コマンドを使用します：

```bash
go run cmd/put-secret/main.go api/external-key "sk-live-abc123xyz789"
```

期待される出力：

```text
[DEBUG] Connecting to Vault at http://localhost:8200
[INFO] Storing secret: api/external-key
[INFO] Successfully stored secret: api/external-key

=== Secret Stored Successfully ===
Key:   api/external-key
Value: sk-live-abc123xyz789

You can retrieve it with:
  go run cmd/get-secret/main.go api/external-key
```

#### ステップ 2: シークレットを取得する

```bash
go run cmd/get-secret/main.go api/external-key
```

期待される出力：

```text
[DEBUG] Connecting to Vault at http://localhost:8200
[INFO] Retrieving secret: api/external-key
[INFO] Successfully retrieved secret: api/external-key (version 1)

=== Secret Retrieved ===
Key:     api/external-key
Value:   sk-live-abc123xyz789
Version: 1
Created: 2025-01-28T10:30:00Z
```

#### ステップ 3: すべてのシークレットを一覧表示

```bash
go run cmd/list-secrets/main.go
```

### STEP 3: 実践例 - API クライアント (10分)

実際のアプリケーションでは、実行時にシークレットを取得し、API 呼び出しに使用する必要があります。この例ではそのパターンを示します：

```bash
# まず、API キーを保存
go run cmd/put-secret/main.go api/external-key "sk-live-abc123xyz789"

# 次に API クライアントを実行
go run cmd/api-client/main.go
```

`api-client` は以下を示します：

1. 起動時に Vault から API キーを取得
2. キーを使用して API 呼び出し（シミュレート）
3. コードにハードコードされた認証情報なし

**コードパターン (`usecase/api_client.go` より):**

```go
func (ac *apiClient) CallAPI(ctx context.Context, secretKey, apiEndpoint string) error {
    // ステップ 1: Vault から API キーを取得
    secret, err := ac.secretManager.RetrieveSecret(ctx, secretKey)
    if err != nil {
        return fmt.Errorf("failed to retrieve API key: %w", err)
    }

    apiKey := secret.Value

    // ステップ 2: キーを使用して API 呼び出し
    statusCode, _, err := ac.httpClient.Get(apiEndpoint, apiKey)
    // ...
}
```

### STEP 4: シークレットのバージョニングとローテーション (10分)

Vault KV v2 はシークレットを自動的にバージョニングします。実際に見てみましょう：

#### ステップ 1: シークレットを更新する

```bash
# バージョン 1 を保存
go run cmd/put-secret/main.go api/payment-gateway "sk-pay-v1-abc"

# バージョン 2 に更新
go run cmd/put-secret/main.go api/payment-gateway "sk-pay-v2-xyz"

# バージョン 3 に更新
go run cmd/put-secret/main.go api/payment-gateway "sk-pay-v3-new"
```

#### ステップ 2: 現在のバージョンを取得

```bash
go run cmd/get-secret/main.go api/payment-gateway
# バージョン 3（最新）が表示される
```

#### ステップ 3: Vault CLI で履歴を確認（コンテナ内コマンド使用）

```bash
# バージョン一覧
podman exec workshop-vault vault kv metadata get secret/api/payment-gateway

# 特定のバージョンを取得（バージョン 1）
podman exec workshop-vault vault kv get -version=1 secret/api/payment-gateway
```

**バージョニングのユースケース:**

- **ロールバック**: 新しいシークレットが間違っている場合、以前のバージョンに戻す
- **監査**: 誰が何をいつ変更したかを追跡
- **ローテーション**: 古いシークレットを一時的にアクセス可能なまま新しいシークレットをデプロイ

### STEP 5: アーキテクチャの理解 (5分)

コード構造を確認します：

```bash
# Domain Layer（純粋な Go、外部依存なし）
cat domain/secret.go
cat domain/repository.go

# Usecase Layer（ビジネスロジック）
cat usecase/secret_manager.go

# Infra Layer（Vault アダプター）
cat infra/vault/secret.go
```

**重要な観察:** `usecase` レイヤーは自分が Vault と通信していることを知りません。`domain` レイヤーで定義された `SecretRepository` インターフェースについてのみ知っています。これは、ビジネスロジックを一切変更せずに、Vault を AWS Secrets Manager や Azure Key Vault に交換できることを意味します。

### STEP 6: テストによる検証

単体テストと統合テストを実行して、システムの動作を確認します。

```bash
# 単体テスト（モック使用、Vault 不要）
go test ./usecase/... -v

# 統合テスト（Vault コンテナ必要）
go test ./infra/vault/... -v
```

---

## Clean Architecture のポイント

### Domain Layer (`domain/repository.go`)

外部依存のない純粋な Go インターフェース：

```go
type SecretRepository interface {
    GetSecret(ctx context.Context, key string) (*Secret, error)
    PutSecret(ctx context.Context, key, value string) error
    ListSecrets(ctx context.Context) ([]SecretMetadata, error)
}
```

### Infra Layer (`infra/vault/secret.go`)

Vault 固有の実装：

```go
type VaultSecretRepository struct {
    client *api.Client
    path   string // KV v2 マウントパス
}

func (r *VaultSecretRepository) GetSecret(ctx context.Context, key string) (*domain.Secret, error) {
    secret, err := r.client.KVv2(r.path).Get(ctx, key)
    // Vault レスポンスを domain.Secret にマッピング
}
```

### Usecase Layer (`usecase/secret_manager.go`)

Vault について知らないビジネスロジック：

```go
type secretManager struct {
    repo domain.SecretRepository
}

func (sm *secretManager) RetrieveSecret(ctx context.Context, key string) (*domain.Secret, error) {
    return sm.repo.GetSecret(ctx, key)
}
```

---

## 環境変数

アプリケーションは以下の環境変数を適切なデフォルト値と共に使用します：

| 変数           | デフォルト                | 説明                       |
|----------------|---------------------------|----------------------------|
| `VAULT_ADDR`   | `http://localhost:8200`   | Vault サーバーのアドレス   |
| `VAULT_TOKEN`  | `dev-workshop-token`      | Vault 認証トークン         |

実行時に上書きできます：

```bash
VAULT_ADDR=http://vault.example.com:8200 \
VAULT_TOKEN=your-token-here \
go run cmd/get-secret/main.go api/key
```

---

## 片付け

実習が終了したら：

```bash
make vault-down
```

これにより Vault コンテナが停止・削除されます。注：Dev モードのデータは一時的であり、コンテナが停止すると失われます。

---

## 次のステップ

この実習を完了した後、以下を検討してください：

1. **本番デプロイ**: Vault 本番硬化（TLS、Raft ストレージ、自動アンシール）について学ぶ
2. **動的シークレット**: 自動ローテーションされるデータベース認証情報を探索
3. **Transit シークレットエンジン**: データを保存せずに暗号化・復号化
4. **ID**: Vault の認証方法（GitHub、AWS、Kubernetes）を使用
5. **監査ログ**: コンプライアンスのための詳細な監査証跡を有効化

---

## 参考文献

- [HashiCorp Vault ドキュメント](https://developer.hashicorp.com/vault/docs)
- [Vault Go API クライアント](https://github.com/hashicorp/vault/api)
- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)