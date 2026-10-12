# クリーンアーキテクチャ実習 (WS3): 通信プロトコルの差し替え

> **⏱️ 所要時間**: 約 90 分
> **🎯 レベル**: 中級
> **💻 環境**: Go 1.26.2+、HTTP → gRPC、Linux / macOS / WSL2

## ゴール

- HTTP と gRPC の入出力を同じ UseCase へ接続する。
- 変更が Presentation Adapter と Composition Root に収まることを差分で確認する。

## 前提条件

- [ ] Go 1.26.2+、C コンパイラー（SQLite の CGO）、Python 3、curl、grpcurl
- [ ] WS1・WS2 修了、または Port と依存注入を理解している
- [ ] 8080 と 9090 が空いている
- コードを再生成する場合のみ protoc と生成プラグインが必要。生成済み Go コードは同梱している。

## 開始状態と完成状態

**開始:** HTTP + SQLite、スレ主限定モードを含む BBS。**完成:** 同じ機能を gRPC の `127.0.0.1:9090` へ公開する。WS3 は独立した演習であり、WS4 へこのコピーを持ち越さない。

開始用コピーを作るときはリポジトリルートで実行します。既存コピーは上書きされないため、再実行には別の名前を使います。

```bash
# ターミナル B — リポジトリルート
export WORKSHOP_REPO="$PWD"
python3 tools/prepare_bbs.py ws3 /tmp/bbs-ws3
cd /tmp/bbs-ws3
go test ./...
# 期待される結果: Prepared ws3 ...、既存テストが成功
```

## 実習ステップ

### STEP 1: HTTP 版を確認する

```bash
# ターミナル A
cd /tmp/bbs-ws3
go run ./cmd/bbs
# 期待される結果: server starting addr=:8080
```

```bash
# ターミナル B — /tmp/bbs-ws3
curl -i http://localhost:8080/api/boards
# 期待される結果: 200 OK、boards 内に programming / news / chat
```

### ✅ チェックポイント: 基準状態

- [ ] HTTP 経由で板一覧を取得した
- [ ] gRPC へ変えるとき、どのファイルが変わるか先に予測した

### STEP 2: Proto と gRPC Adapter を追加する

まず [完全な Proto](assets/bbs/api/proto/bbs.proto) と [完全な Adapter](assets/bbs/internal/adapters/presentation/grpc/server.go) を読みます。入力は UseCase DTO へ、出力は Proto へ変換しています。Domain Error は gRPC status へ変換し、未知のエラー詳細は公開しません。

**そのまま実行:** 参照実装をコピーします。自分で実装する場合も、5 つの RPC と DTO 変換を揃えてから同じ検証を行います。

```bash
# ターミナル B — /tmp/bbs-ws3
mkdir -p api/proto
cp "$WORKSHOP_REPO/software/assets/bbs/api/proto/bbs.proto" api/proto/
cp -R "$WORKSHOP_REPO/software/assets/bbs/internal/adapters/presentation/grpc" internal/adapters/presentation/
mkdir -p /tmp/bbs-ws3-check
go build -o /tmp/bbs-ws3-check/ ./...
# 期待される結果: コンパイルエラーなし（複数 main があるため出力先にはディレクトリを指定）
```

生成済み `pb/` はこの Proto に対応します。再生成は [BBS README](assets/bbs/README_ja.md) の固定バージョン手順を使ってください。

### STEP 3: Composition Root を切り替える

ターミナル A を Ctrl+C で停止します。次は既存 `main.go` の完全な置換です。元の配線と [gRPC 用 Composition Root](assets/bbs/cmd/bbs-grpc/main.go) を比較してからコピーします。

```bash
# ターミナル B — /tmp/bbs-ws3
cp "$WORKSHOP_REPO/software/assets/bbs/cmd/bbs-grpc/main.go" cmd/bbs/main.go
go test ./...
# 期待される結果: すべて成功
```

```bash
# ターミナル A — /tmp/bbs-ws3
go run ./cmd/bbs
# 期待される結果: gRPC server starting addr=127.0.0.1:9090
```

### STEP 4: 同じ機能を gRPC で確認する

```bash
# ターミナル B — /tmp/bbs-ws3
grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards
# 期待される結果: boards 内に programming / news / chat
grpcurl -plaintext -d '{"board_name":"programming","title":"gRPC","author":"gopher","body":"hello"}' localhost:9090 bbs.BBSService/CreateThread
# 期待される結果: thread と post（number: 1）を含む JSON
grpcurl -plaintext -d '{"board_name":"missing","title":"x","body":"x"}' localhost:9090 bbs.BBSService/CreateThread
# 期待される結果: Code: NotFound（このコマンドは非ゼロ終了が正常）
diff -ru .baseline/internal/domain internal/domain
diff -ru .baseline/internal/usecase internal/usecase
diff -ru .baseline/internal/adapters/infra internal/adapters/infra
# 期待される結果: 3 コマンドとも差分なし
```

### ✅ チェックポイント: 完成

- [ ] 板一覧・スレッド作成が gRPC から成功し、不在の板は NotFound
- [ ] Domain・UseCase・Infra の差分がなく、初めの予測と比較した
- [ ] コード内の `transportError` が HTTP のステータス変換に対応することを説明できる

## 学んだこと

- Proto は外側の通信 DTO、UseCase DTO は内側のアプリケーション境界。
- 通信方式の変更には Presentation Adapter と Composition Root を変更する。
- RPC の入出力とエラー変換を揃えることで同じ UseCase を再利用する。

次は [WS4](clean_arch_ws4_ja.md) を新しい HTTP コピーから始めます。

## トラブルシューティング

| 症状                     | 確認と対処                                                  |
| ------------------------ | ----------------------------------------------------------- |
| 9090 に接続できない      | ターミナル A の起動ログを確認し、HTTP 版を停止したか確認    |
| Proto の型が見つからない | Adapter と生成済み pb ディレクトリを両方コピー              |
| WORKSHOP_REPO が空       | 準備したターミナル B を使うか、リポジトリの絶対パスを再設定 |

## 片付け

ターミナル A を Ctrl+C で停止し、ターミナル B でコピーを削除します。

```bash
cd /tmp
rm -r /tmp/bbs-ws3
rm -rf /tmp/bbs-ws3-check
# 期待される結果: 実習コピーとビルド成果物を削除。原本は変更なし
```

## 参考文献

- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers Go Guide](https://protobuf.dev/reference/go/go-generated/)
- [Clean Architecture](clean_arch_ja.md)
