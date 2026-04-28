# クリーンアーキテクチャ実習 (WS4): ビジネスルールの追加

この実習では、BBS（2 ちゃんねる風掲示板）に「スレ主しか書き込めない」というビジネスルールを追加します。
**内側から外側へ変更が波及する様子**を体験し、各層の変更がその層の責務に直結する最小限のものであることを確認します。

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。

## 実習のシナリオ

「特定のスレッドでは、スレ主（1 レス目の投稿者）しか書き込めない」という制限モードを追加します。

---

## 課題: スレ主のみ書き込み可能モードの追加

### 変更範囲の全体像

ビジネスルールの追加は内側から外側へ波及しますが、各層の変更は **その層の責務に直結する最小限のもの** です。

| 層 | 変更内容 | 役割 |
| ---- | --------- | ------ |
| **Domain** | `Thread.OwnerOnly` フラグ追加、`CanPost()` メソッド追加、`ErrNotThreadOwner` エラー追加 | ルールの定義 |
| **UseCase** | `thread.CanPost(in.Author)` の呼び出しを1行追加、スレッド作成時に `Owner` を設定 | ルールの適用 |
| **Infra** | `threads` テーブルに `owner_only` 列を追加、読み書きを対応 | 永続化の追従 |
| **Framework** | エラー変換に `ErrNotThreadOwner → 403 Forbidden` を1行追加、`createThreadRequest` に `owner_only` フィールドを追加 | 表示と入力の追従 |

### Step 1: Domain 層 — ルールの定義

ビジネスルールを Entity にカプセル化します。

**1-1. Thread Entity にフラグと判定メソッドを追加する**

> **注意**: 現在の `Thread` 構造体には `OwnerOnly` と `Owner` フィールドは存在しません。このステップで新規に追加します。

```go
// domain/entity/thread.go
type Thread struct {
    ID           int64
    BoardID      int64
    Title        string
    PostCount    int
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
// usecase/post_usecase.go
func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    var out *CreatePostOutput
    if err := u.tm.RunInTransaction(ctx, func(txCtx context.Context) error {
        thread, err := u.threadRepo.FindByID(txCtx, in.ThreadID)
        if err != nil {
            return err
        }

        // ↓ この1行を追加するだけ
        if !thread.CanPost(in.Author) {
            return domain.ErrNotThreadOwner
        }

        count, err := u.postRepo.CountByThreadID(txCtx, thread.ID)
        if err != nil {
            return err
        }
        post, err := entity.NewPost(thread.ID, count+1, in.Author, in.Body, in.Sage)
        if err != nil {
            return err
        }

        if err := u.postRepo.Save(txCtx, post); err != nil {
            return err
        }
        thread.Bump(post.CreatedAt, in.Sage)
        if err := u.threadRepo.Save(txCtx, thread); err != nil {
            return err
        }

        out = &CreatePostOutput{Post: toPostDTO(post)}
        return nil
    }); err != nil {
        return nil, err
    }

    return out, nil
}
```

**2-1.1. トランザクション管理について**

UseCase 層はトランザクション境界を制御する責務も持ちます。

- **TransactionManager**: Domain Port として定義される（`port.TransactionManager`）
- **Infra Adapter**: SQLite の `sql.Tx` などを使って具象実装
- **UseCase**: 複数のリポジトリ操作を 1 つのトランザクションにまとめる

```go
// domain/port/transaction.go
type TransactionManager interface {
    RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

投稿処理では、スレッド取得・権限判定・投稿数カウント・番号採番・投稿保存・スレッド更新（Bump）の全工程をアトミックに実行する必要があります。どの手順が失敗しても全変更を巻き戻すため、UseCase がトランザクション境界を制御します（例: 投稿だけ保存されてスレッドが更新されない状態を防ぐ）。

**確認ポイント**: トランザクションは「技術的な詳細」ではなく「アプリケーションポリシー」です。UseCase が「この操作セットはアトミックであるべき」と判断し、Infra Adapter がその具体的な実装（`BEGIN`/`COMMIT`/`ROLLBACK`）を提供します。

**2-2. DTO にフラグを追加する**

`CreateThreadInput`（`usecase/dto.go`）に `OwnerOnly` フィールドを追加します。これにより Framework 層からフラグを渡せるようになります。

```go
// usecase/dto.go
type CreateThreadInput struct {
    BoardName string
    Title     string
    Author    string
    Body      string
    OwnerOnly bool   // 追加: スレ主限定モード
}
```

**2-3. Owner はいつ設定されるか**

`Owner`（スレ主 = 1 番目の投稿者）は、スレッド作成時の `CreateThreadUseCase` で設定します。

```go
// usecase/thread_usecase.go（CreateThreadUseCase.Execute 内）
thread, err := entity.NewThread(board.ID, in.Title)
if err != nil {
    return nil, err
}
thread.OwnerOnly = in.OwnerOnly  // Framework から渡されたフラグ
thread.Owner = in.Author         // 1番目の投稿者をスレ主として記録
```

### Step 3: Infra 層 — 永続化の追従

DB に新しい列を追加し、読み書きに対応します。**SQL とモデル変換だけの変更**です。

**3-1. マイグレーション**

```sql
ALTER TABLE threads ADD COLUMN owner_only BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN owner TEXT NOT NULL DEFAULT '';
UPDATE threads SET owner_only = FALSE, owner = '';
```

**3-2. DB モデルの更新**

```go
// infra/persistence/model.go
type ThreadModel struct {
    ID            int64  `db:"id"`
    BoardID       int64  `db:"board_id"`
    Title         string `db:"title"`
    PostCount     int    `db:"post_count"`
    OwnerOnly     bool   `db:"owner_only"`  // 追加
    Owner         string `db:"owner"`       // 追加
    // ...
}
```

**3-3. リポジトリの変換ロジック更新**

```go
// infra/persistence/thread_repo.go
func (r *ThreadRepository) toEntity(m *ThreadModel) *entity.Thread {
    return &entity.Thread{
        ID:           m.ID,
        BoardID:      m.BoardID,
        Title:        m.Title,
        PostCount:    m.PostCount,
        OwnerOnly:    m.OwnerOnly,  // 追加
        Owner:        m.Owner,      // 追加
        // ...
    }
}
```

**3-4. ドライバエラーのドメインエラーへの変換**

Infra Adapter は、データベースドライバからのエラーをドメインエラーに変換する**責務を持ちます**。これにより、UseCase 層はデータベースの詳細を知らなくて済みます。

```go
// infra/persistence/thread_repo.go
import (
    "database/sql"
    "errors"
    "yourproject/domain"
)

func (r *ThreadRepository) FindByID(ctx context.Context, id int64) (*entity.Thread, error) {
    var m ThreadModel
    err := r.db.QueryRowContext(ctx, "SELECT ... FROM threads WHERE id = ?", id).Scan(
        &m.ID, &m.BoardID, &m.Title, &m.PostCount, &m.OwnerOnly, &m.Owner, /* ... */,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            // ドライバエラーをドメインエラーに変換
            return nil, domain.ErrThreadNotFound
        }
        return nil, err
    }
    return r.toEntity(&m), nil
}
```

**確認ポイント**: `sql.ErrNoRows` という技術的なエラーが、UseCase 層へ渡る前に `domain.ErrThreadNotFound` というドメインの概念に変換されています。UseCase は「SQL の結果が空だった」ことを知る必要がありません。

### Step 4: Framework 層 — 表示の追従

エラーハンドリングに 1 case を追加します。

```go
// internal/framework/http/handler/post_handler.go — CreatePost 内のエラーハンドリング
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    out, err := h.createPost.Execute(r.Context(), input)
    // ...既存のエラーハンドリング
    // ↓ 1 case 追加
    case errors.Is(err, domain.ErrNotThreadOwner):
        writeError(w, http.StatusForbidden, err.Error())
    }
}
```

また、スレッド作成時のリクエスト DTO にも `owner_only` フィールドを追加します。

```go
// internal/framework/http/handler/thread_handler.go
type createThreadRequest struct {
    Title     string `json:"title"`
    Author    string `json:"author"`
    Body      string `json:"body"`
    OwnerOnly bool   `json:"owner_only"`  // 追加
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

招待ユーザーを `Thread` エンティティ内で管理する場合（例: `InvitedUsers` フィールド）、Domain の変更だけで済み、UseCase・Infra・Framework は **一切変更不要** です。ただし、招待ユーザーの管理に別テーブルや外部サービスが必要な場合は、Infra や UseCase の変更も発生します。

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

- 「スレ主 = 1 レス目の投稿者」というルールが **SQL** に埋まっている
- 「スレ主以外禁止」というルールが **HTTP handler** に埋まっている
- 「招待された人も OK」に変更する場合、**どこを直すべきか** が不明

---

## この実習のポイント

1. **ルールの居場所が明確**:
    - Domain は「ルールとは何か」を知っている（`CanPost` の中身）
    - UseCase は「いつルールを適用するか」を知っている（呼び出しタイミング）
    - Framework は「ルール違反をどう表示するか」を知っている（403 Forbidden）
    - Infra は「ルールに必要なデータをどう保存するか」を知っている（owner_only 列）
2. **内→外への波及**: ビジネスルールの変更は Domain から始まり、外側に波及するが、各層の変更は自身の責務に限定される。
3. **変更の最小化**: 新しい要件「招待者も OK」が追加されても、`CanPost()` の中身だけを変えれば済む。
