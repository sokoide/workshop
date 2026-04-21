# クリーンアーキテクチャ実習 (WS4): ビジネスルールの追加

この実習では、BBS（2ちゃんねる風掲示板）に「スレ主しか書き込めない」というビジネスルールを追加します。
**内側から外側へ変更が波及する様子**を体験し、各層の変更がその層の責務に直結する最小限のものであることを確認します。

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。

## 実習のシナリオ

「特定のスレッドでは、スレ主（1レス目の投稿者）しか書き込めない」という制限モードを追加します。

---

## 課題: スレ主のみ書き込み可能モードの追加

### 変更範囲の全体像

ビジネスルールの追加は内側から外側へ波及しますが、各層の変更は **その層の責務に直結する最小限のもの** です。

| 層 | 変更内容 | 役割 |
|----|---------|------|
| **Domain** | `Thread.OwnerOnly` フラグ追加、`CanPost()` メソッド追加、`ErrNotThreadOwner` エラー追加 | ルールの定義 |
| **UseCase** | `thread.CanPost(in.Author)` の呼び出しを1行追加 | ルールの適用 |
| **Infra** | `threads` テーブルに `owner_only` 列を追加、読み書きを対応 | 永続化の追従 |
| **Framework** | エラー変換に `ErrNotThreadOwner → 403 Forbidden` を1行追加 | 表示の追従 |

### Step 1: Domain 層 — ルールの定義

ビジネスルールを Entity にカプセル化します。

**1-1. Thread Entity にフラグと判定メソッドを追加する**

```go
// domain/entity/thread.go
type Thread struct {
    ID           int64
    BoardSlug    string
    Title        string
    ResponseCount int
    CreatedAt    time.Time
    LastPostedAt time.Time
    // ↓ 追加
    OwnerOnly bool   // スレ主のみ書き込み可能か
    Owner     string // スレ主（1レス目の投稿者）
}

// 追加: 書き込み可能かを判定するビジネスルール
func (t *Thread) CanPost(author string) bool {
    if !t.OwnerOnly {
        return true // 制限モードでなければ誰でもOK
    }
    return t.Owner == author
}
```

**確認ポイント**: 判定ロジックは `CanPost()` に **1箇所だけ** 存在します。DB も HTTP も知りません。

**1-2. ドメインエラーを追加する**

```go
// domain/error.go
var (
    ErrBoardNotFound  = errors.New("board not found")
    ErrThreadNotFound = errors.New("thread not found")
    // ↓ 追加
    ErrNotThreadOwner = errors.New("only thread owner can post")
)
```

### Step 2: UseCase 層 — ルールの適用

UseCase は「いつルールを適用するか」を知る層です。投稿処理の中に **1行** を追加します。

```go
// usecase/create_post.go
func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    thread, err := u.threadRepo.FindByID(ctx, in.ThreadID)
    if err != nil {
        return nil, err
    }

    // ↓ この1行を追加するだけ
    if !thread.CanPost(in.Author) {
        return nil, domain.ErrNotThreadOwner
    }

    // ...以降のロジックは変更なし
    post := entity.NewPost(thread.ID, in.Author, in.Body, in.Sage)
    // ...
}
```

### Step 3: Infra 層 — 永続化の追従

DB に新しい列を追加し、読み書きに対応します。**SQL とモデル変換だけの変更**です。

**3-1. マイグレーション**

```sql
ALTER TABLE threads ADD COLUMN owner_only BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE threads SET owner_only = FALSE;
```

**3-2. DB モデルの更新**

```go
// infra/persistence/model.go
type ThreadModel struct {
    ID            int64  `db:"id"`
    BoardSlug     string `db:"board_slug"`
    Title         string `db:"title"`
    OwnerOnly     bool   `db:"owner_only"`  // 追加
    // ...
}
```

**3-3. リポジトリの変換ロジック更新**

```go
// infra/persistence/thread_repo.go
func (r *ThreadRepository) toEntity(m *ThreadModel) *entity.Thread {
    return &entity.Thread{
        ID:           m.ID,
        BoardSlug:    m.BoardSlug,
        Title:        m.Title,
        OwnerOnly:    m.OwnerOnly,  // 追加
        Owner:        m.Owner,       // 既存フィールドを活用
        // ...
    }
}
```

### Step 4: Framework 層 — 表示の追従

エラーハンドリングに1 case を追加します。

```go
// framework/handler/post_handler.go — CreatePost 内のエラーハンドリング
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    out, err := h.createPost.Execute(r.Context(), input)
    // ...既存のエラーハンドリング
    // ↓ 1 case 追加
    case errors.Is(err, domain.ErrNotThreadOwner):
        writeError(w, http.StatusForbidden, err.Error())
    }
}
```

### Step 5: 動作確認

```bash
# スレ主限定スレッドを作成
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"スレ主限定スレ","author":"gopher","body":"限定モードON","owner_only":true}'

# スレ主以外が書き込もうとする → 403 Forbidden
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"other","body":"書き込めるかな？"}'

# スレ主が書き込む → 成功
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"gopher","body":"自分のスレだしOK"}'
```

---

## 仕様のさらなる変更にも強い

「スレ主＋指定された人も書き込み可能」に仕様変更されても:

```go
// Domain だけ変更
func (t *Thread) CanPost(author string) bool {
    if !t.OwnerOnly {
        return true
    }
    return t.Owner == author || t.IsInvited(author)
}
```

UseCase、Infra、Framework は **一切変更不要** です。
呼び出し側は `CanPost()` の中身を知らないため、内部実装が変わっても影響しません。

---

## レイヤー分離がない場合との比較

```go
// NG: ビジネスルールが Handler に埋まっている例
func CreatePost(w http.ResponseWriter, r *http.Request) {
    row := db.QueryRow(
        "SELECT author FROM posts WHERE thread_id=? AND number=1", threadID,
    )
    var owner string
    row.Scan(&owner)
    author := r.FormValue("author")
    if owner != author {                    // ← ビジネスルールがここに埋まっている
        w.WriteHeader(403)                  // ← HTTP 依存
        json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
        return
    }
    // INSERT ...
}
```

- 「スレ主 = 1レス目の投稿者」というルールが **SQL** に埋まっている
- 「スレ主以外禁止」というルールが **HTTP handler** に埋まっている
- 「招待された人もOK」に変更する場合、**どこを直すべきか** が不明

---

## この実習のポイント

1. **ルールの居場所が明確**:
    - Domain は「ルールとは何か」を知っている（`CanPost` の中身）
    - UseCase は「いつルールを適用するか」を知っている（呼び出しタイミング）
    - Framework は「ルール違反をどう表示するか」を知っている（403 Forbidden）
    - Infra は「ルールに必要なデータをどう保存するか」を知っている（owner_only 列）
2. **内→外への波及**: ビジネスルールの変更は Domain から始まり、外側に波及するが、各層の変更は自身の責務に限定される。
3. **変更の最小化**: 新しい要件「招待者もOK」が追加されても、`CanPost()` の中身だけを変えれば済む。
