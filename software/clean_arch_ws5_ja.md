# クリーンアーキテクチャ実習 (WS5): 永続化層の差し替え

> **⏱️ 所要時間**: 約 60 分（環境構築を含む目安）
> **🎯 レベル**: 中級
> **💻 環境**: Go 1.26.2+、C コンパイラー、Podman、PostgreSQL 16、Python 3、curl

## ゴール

SQLite を PostgreSQL へ差し替え、Domain・UseCase・HTTP Adapter を変更せずに、板一覧・スレッド作成・スレ主限定モードを確認します。

## 前提条件

- [ ] WS4 の投稿ルールとトランザクション境界を理解している
- [ ] Go 1.26.2+、C コンパイラー（開始状態の SQLite）、Python 3、curl
- [ ] Podman が起動し、8080 と 55432 が空いている。以下の podman コマンドは Docker でも実行可能。

## 開始状態と完成状態

**開始:** WS4 完了相当の HTTP + SQLite。**完成:** 同じ HTTP API + PostgreSQL。WS3 の gRPC 版は持ち越しません。コピーには Go の依存が固定されているので、バージョンを選び直す必要はありません。

```bash
# ターミナル B — リポジトリルート
export WORKSHOP_REPO="$PWD"
python3 tools/prepare_bbs.py ws5 /tmp/bbs-ws5
cd /tmp/bbs-ws5
go test ./...
# 期待される結果: Prepared ws5 ...、既存テストが成功
```

## 実習ステップ

### STEP 1: SQLite の基準状態を確認する

```bash
# ターミナル A
cd /tmp/bbs-ws5
go run ./cmd/bbs
# 期待される結果: :8080 で起動
```

```bash
# ターミナル B — /tmp/bbs-ws5
curl -i http://localhost:8080/api/boards
# 期待される結果: 200 OK、boards 内に programming / news / chat
```

ターミナル A を Ctrl+C で停止します。

### STEP 2: PostgreSQL Adapter を揃える

**予測:** 同じ Port を満たすなら、どの層が変わるでしょうか？回答してから [SQLite 実装](assets/bbs/internal/adapters/infra/persistence/sqlite/) と [完全な PostgreSQL 実装](assets/bbs/internal/adapters/infra/persistence/postgres/) を比較します。

| ファイル                                      | 学ぶ差分                                                             |
| --------------------------------------------- | -------------------------------------------------------------------- |
| board_repo.go / thread_repo.go / post_repo.go | `?` → `$1`、`RETURNING id`、bool の保存、エラー変換                  |
| model.go                                      | DB モデルから Entity への変換                                        |
| transaction.go                                | UseCase の Port を実装し、全 Repository が同じトランザクションへ参加 |
| db.go / schema.sql                            | pgx ドライバー登録、接続確認、完全な DDL、初期の板                   |

`FindByID` の `FOR UPDATE` は投稿番号割り当ての並行競合を避けるためのロックです。トランザクションの context は Infra 内で解釈し、UseCase は `sql.Tx` を知りません。

**そのまま実行:** 完成ファイルをコピーします。自分で書く場合は、上表のファイルと各 Repository Port の全メソッドを実装してから検証します。

```bash
# ターミナル B — /tmp/bbs-ws5
cp -R "$WORKSHOP_REPO/software/assets/bbs/internal/adapters/infra/persistence/postgres" internal/adapters/infra/persistence/
go test ./...
# 期待される結果: ビルドを含めて成功。接続テストは BBS_TEST_DATABASE_URL 未設定なら SKIP
```

### STEP 3: DB を起動し、DDL と初期データを適用する

コンテナはこの実習専用です。同名の既存コンテナがある場合は別名を使い、以降のコマンドも揃えてください。

```bash
# ターミナル B — /tmp/bbs-ws5
podman run -d --name bbs-ws5-postgres -p 127.0.0.1:55432:5432   -e POSTGRES_USER=bbs -e POSTGRES_PASSWORD=bbs -e POSTGRES_DB=bbs postgres:16
# 期待される結果: コンテナ ID
for attempt in $(seq 1 30); do
  podman exec bbs-ws5-postgres pg_isready -U bbs -d bbs && break
  sleep 1
done
podman exec bbs-ws5-postgres pg_isready -U bbs -d bbs
# 期待される結果: accepting connections（準備できなければ先へ進まない）
podman exec -i bbs-ws5-postgres psql -v ON_ERROR_STOP=1 -U bbs -d bbs < internal/adapters/infra/persistence/postgres/schema.sql
# 期待される結果: CREATE TABLE と INSERT、エラーなし
podman exec bbs-ws5-postgres psql -U bbs -d bbs -c 'SELECT name FROM boards ORDER BY name;'
# 期待される結果: chat / news / programming の 3 行
```

`OpenDB` も同じ埋め込み DDL をトランザクションで適用します。再起動時に板が重複しない構成です。TLS を省いた接続文字列は、このローカル演習の DB 用です。

### STEP 4: Composition Root を差し替える

**既存ファイルの完全な置換:** [PostgreSQL 用 main.go](assets/bbs/cmd/bbs-postgres/main.go) を読んで、Repository と TransactionManager の配線を確認します。UseCase と HTTP Handler の組み立ては同じです。

```bash
# ターミナル B — /tmp/bbs-ws5
cp "$WORKSHOP_REPO/software/assets/bbs/cmd/bbs-postgres/main.go" cmd/bbs/main.go
go build -o bbs ./cmd/bbs
# 期待される結果: 出力なし、bbs が作成される
BBS_TEST_DATABASE_URL='postgres://bbs:bbs@127.0.0.1:55432/bbs?sslmode=disable' go test -race ./internal/adapters/infra/persistence/postgres -v
# 期待される結果: 接続・保存・エラー変換・ロールバックのテストが PASS
```

```bash
# ターミナル A — /tmp/bbs-ws5
DATABASE_URL='postgres://bbs:bbs@127.0.0.1:55432/bbs?sslmode=disable' ./bbs
# 期待される結果: 127.0.0.1:8080 で起動
```

### STEP 5: 同じ API とルールを検証する

```bash
# ターミナル B — /tmp/bbs-ws5
curl -i http://localhost:8080/api/boards
# 期待される結果: 200 OK、SQLite 版と同じ boards の JSON 形式
THREAD_ID=$(curl -fsS -X POST http://localhost:8080/api/boards/programming/threads -H 'Content-Type: application/json' -d '{"title":"Postgres","author":"owner","body":"first","owner_only":true}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["thread"]["id"])')
# 期待される結果: 新しいスレッド ID
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"other","body":"denied"}'
# 期待される結果: 403 Forbidden
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"owner","body":"allowed"}'
# 期待される結果: 201 Created、number: 2
diff -ru .baseline/internal/domain internal/domain
diff -ru .baseline/internal/usecase internal/usecase
diff -ru .baseline/internal/adapters/presentation internal/adapters/presentation
# 期待される結果: 3 コマンドとも差分なし
```

### ✅ チェックポイント

- [ ] 完全な PostgreSQL Adapter が全 Port を満たし、接続テストが成功した
- [ ] 板一覧・作成・拒否・許可を同じ HTTP API で確認した
- [ ] Domain・UseCase・Presentation は変更せず、初めの予測と比較した

## 学んだこと

- DB 固有の SQL とドライバーを Infra に閉じ込める。
- DB と全 Repository のトランザクションを揃え、投稿ルールを保つ。
- スキーマ、初期データ、接続確認も差し替えに必要な作業である。

次は [WS6](clean_arch_ws6_ja.md) の専用コピーから開始できます。

## トラブルシューティング

| 症状                    | 対処                                                          |
| ----------------------- | ------------------------------------------------------------- |
| connection refused      | コンテナの pg_isready と 55432 の公開を確認                   |
| unknown driver pgx      | db.go の stdlib 登録と go.mod を確認。完全な Adapter をコピー |
| relation does not exist | psql の ON_ERROR_STOP 出力と schema.sql の適用を確認          |
| 期待する板がない        | schema.sql の初期データを確認                                 |

## 片付け

ターミナル A を Ctrl+C で停止してから実行します。

```bash
# ターミナル B
podman rm -f bbs-ws5-postgres
cd /tmp
rm -r /tmp/bbs-ws5
# 期待される結果: 専用 DB コンテナ、コピー、SQLite DB を削除
```

## 参考文献

- [pgx database/sql adapter](https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib)
- [PostgreSQL CREATE TABLE](https://www.postgresql.org/docs/16/sql-createtable.html)
- [Clean Architecture](clean_arch_ja.md)
