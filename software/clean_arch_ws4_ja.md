# クリーンアーキテクチャ実習 (WS4): ビジネスルールの追加

> **⏱️ 所要時間**: 約 60 分
> **🎯 レベル**: 中級
> **💻 環境**: Go 1.26.2+、curl、Linux / macOS / WSL2

## ゴール

- スレ主限定モードの投稿ルールを Domain に実装し、UseCase から呼ぶ。
- 許可・拒否の結果と、失敗時に投稿が保存されないことを確認する。

## 前提条件

- [ ] Go 1.26.2+、C コンパイラー、Python 3、curl
- [ ] WS1・WS2 のレイヤーと Port を理解している。WS3 は推奨だが、gRPC のコードは持ち越さない。

## 開始状態と完成状態

**開始:** HTTP + SQLite。Owner/OwnerOnly の Entity フィールド、DTO、DB 列、HTTP の 403 変換は足場として実装済み。`CanPost` の判定と UseCase の拒否処理は TODO で、他人の投稿をまだ許可する。

**完成:** 限定モードではスレ主だけ許可する。既存アセットは完成例なので、開始用コピーを使う。今回は足場を再作成せず、ルールと呼び出しの 2 か所に集中する。

```bash
# ターミナル B — リポジトリルート
export WORKSHOP_REPO="$PWD"
python3 tools/prepare_bbs.py ws4 /tmp/bbs-ws4
cd /tmp/bbs-ws4
go test ./...
# 期待される結果: 開始状態のテストは成功。拒否ルールのテストは次の STEP で追加
```

## 実習ステップ

### STEP 1: 拒否ルールのテストを先に追加する

**完全なテストファイル:** 完成例からコピーし、未実装の仕様を確認します。

```bash
# ターミナル B — /tmp/bbs-ws4
cp "$WORKSHOP_REPO/software/assets/bbs/internal/domain/entity/thread_test.go" internal/domain/entity/
cp "$WORKSHOP_REPO/software/assets/bbs/internal/usecase/post_usecase_test.go" internal/usecase/
go test ./internal/domain/entity ./internal/usecase
# 期待される結果: 他人の投稿拒否に関するテストが FAIL（この段階は非ゼロ終了が正常）
```

### STEP 2: Domain の判定を実装する

**既存メソッドの置換:** `internal/domain/entity/thread.go` の TODO を含む `CanPost` を次へ変更します。

```go
func (t *Thread) CanPost(author string) bool {
    if !t.OwnerOnly {
        return true
    }
    return t.Owner == author
}
```

### STEP 3: UseCase で保存前に拒否する

**既存ファイルの差分:** `internal/usecase/post_usecase.go` の TODO を次へ置き換えます。

```go
if !thread.CanPost(in.Author) {
    return domain.ErrNotThreadOwner
}
```

判定は投稿数取得・保存より前、トランザクション内に置きます。HTTP では既存の Handler がこのエラーを 403 に変換します。

```bash
go test ./...
# 期待される結果: Domain と UseCase を含む全テストが成功
```

### ✅ チェックポイント: ルール

- [ ] 限定モードでない場合は誰でも投稿できる
- [ ] 限定モードでは同じ投稿者名だけ許可され、拒否時に投稿を保存しない

### STEP 4: HTTP で確認する

```bash
# ターミナル A
cd /tmp/bbs-ws4
go run ./cmd/bbs
# 期待される結果: :8080 で起動
```

```bash
# ターミナル B — /tmp/bbs-ws4
THREAD_ID=$(curl -fsS -X POST http://localhost:8080/api/boards/programming/threads -H 'Content-Type: application/json' -d '{"title":"Owner only","author":"owner","body":"first","owner_only":true}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["thread"]["id"])')
# 期待される結果: THREAD_ID に新しく作ったスレッドの ID
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"other","body":"denied"}'
# 期待される結果: 403 Forbidden
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"owner","body":"allowed"}'
# 期待される結果: 201 Created、number: 2（拒否した投稿は数えない）
```

### ✅ チェックポイント: 動作と理解

- [ ] 他人の投稿は 403、スレ主は 201、投稿番号は 2
- [ ] なぜ HTTP Handler だけでルールを判断しないのか説明できる

<details>
<summary>ヒントと解答例</summary>

Domain にルールを置くことで、gRPC や CLI からも同じ判定を使えます。UseCase は保存前に判定を呼び、Presentation はエラーを通信方式へ変換します。ここでの投稿者名の比較は教材上の識別であり、認証された身元の保証ではありません。認証は WS7 で扱います。

</details>

## 学んだこと

- Domain に判断、UseCase に判断を呼ぶ順序、Presentation にエラー変換を置く。
- 先に失敗するテストで仕様を確かめ、実装後の成功を確認する。
- DTO と DB の足場が既にある場合、ルール変更だけを独立して学べる。

次は [WS5](clean_arch_ws5_ja.md) を新しい完成状態の HTTP コピーから開始します。

## トラブルシューティング

| 症状                     | 対処                                              |
| ------------------------ | ------------------------------------------------- |
| 最初から拒否テストが通る | prepare_bbs.py の ws4 開始状態を新しいパスに作る  |
| 404 になる               | 固定 ID ではなく直前の作成結果 THREAD_ID を使う   |
| 他人の投稿が 201         | CanPost と、保存前の UseCase の呼び出しを両方確認 |

## 片付け

ターミナル A を Ctrl+C で停止し、コピーを削除します。

```bash
# ターミナル B
cd /tmp
rm -r /tmp/bbs-ws4
# 期待される結果: コピーと SQLite DB を削除。原本は変更なし
```

## 参考文献

- [Clean Architecture](clean_arch_ja.md)
- [Go Testing](https://pkg.go.dev/testing)
