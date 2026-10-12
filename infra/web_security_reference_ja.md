# Web セキュリティ参考資料

防御実習の背景知識を必要な項目から参照する資料です。実習手順は [防御編](web_security_defense_ja.md) にあります。

## リファレンス: この実習でハンズオン化しなかった背景知識

> 以下のトピックは**リファレンス資料**です。この被害者アプリケーションの構造上ハンズオン化できなかった（アプリがこれらを行使しない）ものですが、対策の全体像を補完します。

### CORS の基礎

CORS（クロスオリジンリソースシェアリング）は、「異なるオリジン（ドメイン・ポート・プロトコル）」のリソースへのアクセスをブラウザが制御する仕組みです。

**オリジンの例:**

```text
https://example.com:443/api/data
       ^^^^^^^^^^^^^^ ^^^
       オリジン         パス

同一オリジン:
  ✓ https://example.com/api/data
  ✓ https://example.com:443/api/data

異なるオリジン:
  ✗ https://api.example.com/api/data  （異なるドメイン）
  ✗ http://example.com/api/data       （異なるプロトコル）
  ✗ https://example.com:8080/api/data （異なるポート）
```

**CORS が必要な理由:** ブラウザには**同一オリジンポリシー (SOP)** があり、クロスオリジンリクエストを制限しています。これがないと、任意のサイトの JavaScript がログイン中のサービスへ `fetch()` してレスポンスを読めてしまいます。CORS は SOP を安全に緩和する仕組みです。

```text
1. ブラウザがリクエストに Origin ヘッダーを追加
   Origin: https://example.com

2. サーバーがレスポンスに Access-Control-Allow-Origin を追加
   Access-Control-Allow-Origin: https://example.com  （許可）
   Access-Control-Allow-Origin: *                        （全て許可）

3. ブラウザが一致を確認
   - 一致 → レスポンスを JavaScript に渡す
   - 不一致 → エラーを投げる
```

**CORS ヘッダーの重要ポイント:**

| ヘッダー                           | 説明                         | 脆弱性                     |
| :--------------------------------- | :--------------------------- | :------------------------- |
| `Access-Control-Allow-Origin`      | 許可するオリジン             | `*` は慎重に               |
| `Access-Control-Allow-Credentials` | Cookie等の送信を許可         | `true` の時は `*` 使用不可 |
| `Access-Control-Allow-Methods`     | 許可するHTTPメソッド         | 必要なメソッドのみ         |
| `Access-Control-Allow-Headers`     | 許可するリクエストヘッダー   | 必要なヘッダーのみ         |
| `Access-Control-Max-Age`           | プリフライトのキャッシュ時間 | 適切な値を設定             |

**プリフライトリクエスト（OPTIONS メソッド）:**

```text
複雑なリクエスト（PUT、DELETE、カスタムヘッダー等）の場合:

1. ブラウザが先に OPTIONS リクエストを送信
   OPTIONS /api/data HTTP/1.1
   Origin: https://example.com
   Access-Control-Request-Method: PUT

2. サーバーが許可を返す
   Access-Control-Allow-Origin: https://example.com
   Access-Control-Allow-Methods: GET, PUT, DELETE

3. ブラウザが実際の PUT リクエストを送信
   PUT /api/data HTTP/1.1
   Origin: https://example.com
```

**CORS と CSRF の関係:**

```text
CORS が強制されない場合:
  攻撃者サイトから fetch() → リクエスト送信可能
                           → Cookie も自動付与
                           → レスポンスを読める
                           → CSRF 型攻撃が容易

CORS が強制される場合 (STEP 6):
  攻撃者サイトから fetch() → リクエストは still 送信される
                           → Cookie も still 付与される（CORS の外）
                           → レスポンスはブロックされる
                           → 攻撃者はレスポンスを読めない

※注意: レスポンスを読めなくても、POST/DELETE等の副作用は発生
   → CSRF トークンや SameSite Cookie は依然として必要
```

### CSRF と CORS の保護目的の違い

|                    | **CSRF**                                     | **CORS**                                     |
| :----------------- | :------------------------------------------- | :------------------------------------------- |
| **主な対象**       | **リクエストの送信**                         | **レスポンスの読み取り**                     |
| **保護対象**       | **サーバー側**（意図しない操作の実行を防ぐ） | **クライアント側**（機密データの盗難を防ぐ） |
| **攻撃の内容**     | なりすまして**操作**させる                   | 別サイトから**データを窃取**する             |
| **ブラウザの役割** | Cookieを自動付与（攻撃を助ける側面）         | レスポンス読み取りをブロック（防御する側面） |

**CSRF の保護目的:** 「誰がリクエストを送ったか」を検証する

```text
攻撃者: ユーザーになりすまして「パスワード変更」リクエストを送りたい
         ↓
サーバー: 本当にユーザー本人の操作か確認する必要がある
         ↓
CSRFトークン: フォームを見たユーザーしか持っていない秘密の値
         ↓
結果: 攻撃者サイトにはトークンがない → リクエスト拒絶 ✓
```

**CORS の保護目的:** 「誰がレスポンスを読めるか」を制御する

```text
攻撃者のJavaScript: fetch('https://victim.com/api/profile')
         ↓
ブラウザ: Cookie付きでリクエストは送信する
         ↓
サーバー: 認証成功 → 機密データを含むレスポンス
         ↓
【ここがCORSの分岐点】
         ↓
CORS不許可: ブラウザがレスポンスをブロック ✓
          → JavaScriptはデータを読めない

CORS誤設定: ブラウザがレスポンスをJavaScriptに渡す ✗
          → 攻撃者が機密データを窃取
```

**相互補完的な関係:**

```text
【CSRF 対策なし】
攻撃者サイトのフォーム → 不正なリクエスト送信 → サーバーが実行
                       → データ変更されてしまう ✗

【CORS 対策なし】
攻撃者サイトの fetch → リクエスト送信 → サーバーが実行
                     → レスポンス読めないが、副作用は発生 ✗

【両方必要】
CSRF: リクエストの真正性を保証（操作の防御）
CORS: レスポンスの読み取りを制限（情報の防御）
```

### プリフライトリクエストだけで CSRF を防げない理由

**結論:** プリフライトリクエスト（OPTIONS メソッド）は CSRF を完全には防げません。

**プリフライトが発生する条件** — 「複雑なリクエスト」の場合のみです。

```javascript
// ✅ プリフライト発生
fetch(url, {
  method: "PUT", // PUT/DELETE/PATCH
  headers: { "X-Custom": "value" }, // カスタムヘッダー
});

// ❌ プリフライトなし（シンプルなリクエスト）
fetch(url, {
  method: "POST",
  headers: { "Content-Type": "application/x-www-form-urlencoded" },
});
```

**シンプルな POST リクエストの条件:**

- メソッド：GET, HEAD, POST のみ
- ヘッダー：`Accept`, `Accept-Language`, `Content-Language`, `Content-Type` のみ
- `Content-Type`：`application/x-www-form-urlencoded`, `multipart/form-data`, `text/plain` のみ

**CSRF 攻撃はフォーム送信を使ってプリフライトを回避します:**

```html
<!-- 攻撃者サイトのHTML -->
<form action="https://victim.com/update-email" method="POST">
  <input type="hidden" name="email" value="hacker@evil.com" />
</form>
<script>
  document.forms[0].submit(); // 自動送信
</script>
```

このリクエストは:

```text
POST /update-email HTTP/1.1
Host: victim.com
Content-Type: application/x-www-form-urlencoded
Cookie: session_id=...  ← 自動付与

email=hacker@evil.com
```

**プリフライトなし**でそのまま送信されます → CORS チェックもバイパスされます。

**攻撃方法ごとの比較:**

| 攻撃方法                     | プリフライト発生？ | CORSで防げる？ | CSRFトークンで防げる？ |
| :--------------------------- | :----------------: | :------------: | :--------------------: |
| **フォーム POST**            |      ❌ なし       |    ❌ 不可     |        ✅ 可能         |
| **fetch + PUT/DELETE**       |      ✅ あり       |    ✅ 可能     |        ✅ 可能         |
| **fetch + カスタムヘッダー** |      ✅ あり       |    ✅ 可能     |        ✅ 可能         |
| **imgタグによるGET**         |      ❌ なし       |    ❌ 不可     |       ⚠️ 限定的        |

**まとめ:**

1. **フォーム送信による攻撃を防げない** — 最も一般的な CSRF ベクトル。プリフライトなしでリクエストが送信される。
2. **GET リクエストによる攻撃を防げない** — `<img src="https://victim.com/delete-account">` など。GET は常にプリフライトなし。
3. **SameSite Cookie がないとリスクが残る** — プリフライトありのリクエストでも Cookie は送信される。読み取りは防げても、副作用（操作実行）は防げない。

したがって、CORS は「情報漏洩対策」としてのみ機能し、CSRF 防御には CSRF トークンまたは SameSite Cookie が必須です。まさに STEP 3 と STEP 4 で構築した重層防御のことです。

### CSP 詳解

#### CSP とは何か？

CSP は、ブラウザに対して「このページで許可するリソース（スクリプト、画像、スタイルなど）の出所」を宣言する HTTP レスポンスヘッダーです。XSS の被害を大幅に軽減できる、強力な防御レイヤーです。

**CSP がない場合:**

```text
ブラウザ: ページ内の <script> タグをすべて実行する
         → 攻撃者が注入した <script>alert(document.cookie)</script> も実行されてしまう
```

**CSP がある場合:**

```text
サーバー: Content-Security-Policy: script-src 'self'
ブラウザ: 「このページで実行してよいスクリプトは、同一オリジン(self)から読み込まれたものだけ」
         → インラインの <script> タグはブロック
         → 攻撃者が注入したスクリプトは実行されない ✓
```

#### CSP の仕組み

```text
1. サーバーが HTTP レスポンスヘッダーでポリシーを宣言
   Content-Security-Policy: script-src 'self'; style-src 'self'

2. ブラウザがページの読み込み時にポリシーを評価
   - 各リソースの読み込み先をチェック
   - ポリシーに違反するリソースはブロック

3. 違反があった場合
   - コンソールにエラーメッセージを出力
   - (設定していれば) 違反レポートを送信
```

#### 主要ディレクティブ一覧

| ディレクティブ    | 制御対象                     | 例                                             |
| :---------------- | :--------------------------- | :--------------------------------------------- |
| `script-src`      | JavaScript の読み込み元      | `'self'`, `'unsafe-inline'`, `'nonce-abc'`     |
| `style-src`       | CSS の読み込み元             | `'self'`, `'unsafe-inline'`                    |
| `img-src`         | 画像の読み込み元             | `'self'`, `https://images.example.com`         |
| `font-src`        | フォントの読み込み元         | `'self'`, `https://fonts.googleapis.com`       |
| `connect-src`     | fetch/XHR/WebSocket の接続先 | `'self'`, `https://api.example.com`            |
| `frame-src`       | iframe の読み込み元          | `'self'`, `https://trusted.com`                |
| `frame-ancestors` | このページを埋め込める親     | `'none'`, `'self'`（クリックジャッキング対策） |
| `default-src`     | 上記のデフォルト値           | `'self'`                                       |
| `object-src`      | Flash/Java 等のプラグイン    | `'none'`（推奨）                               |
| `base-uri`        | `<base>` タグの制限          | `'self'`                                       |
| `form-action`     | フォームの送信先             | `'self'`                                       |

#### ソース値の意味

| ソース値          | 意味                                    | リスク                   |
| :---------------- | :-------------------------------------- | :----------------------- |
| `'self'`          | 同一オリジンのみ許可                    | 安全（推奨）             |
| `'none'`          | 一切許可しない                          | 最も安全                 |
| `'unsafe-inline'` | インラインスクリプト/スタイルを許可     | 危険（XSS 軽減効果なし） |
| `'unsafe-eval'`   | `eval()` 等を許可                       | 危険                     |
| `'nonce-<value>'` | 指定 nonce 属性を持つスクリプトのみ許可 | 安全（推奨）             |
| `'sha256-<hash>'` | ハッシュに一致するスクリプトのみ許可    | 安全                     |
| `https://...`     | 指定 URL のみ許可                       | 中程度                   |
| `*`               | すべて許可                              | 危険                     |

#### 段階的な CSP 導入

**フェーズ1: レポートのみモード（影響なし）**

```go
// Content-Security-Policy-Report-Only はブロックせず、違反をレポートするだけ
w.Header().Set("Content-Security-Policy-Report-Only",
    "default-src 'self'; "+
    "report-uri /csp-report")

// 本番環境でまずこれを設定し、違反ログを収集してから
// 強制モードの CSP に切り替える
```

**フェーズ2: 緩やかなポリシー（移行期）**

```go
// インラインスクリプトを一時的に許可（既存コードの修正中）
w.Header().Set("Content-Security-Policy",
    "default-src 'self'; "+
    "script-src 'self' 'unsafe-inline'; "+ // 段階的に削除
    "style-src 'self' 'unsafe-inline'; "+
    "img-src 'self' data:; "+
    "frame-ancestors 'none'")
```

**フェーズ3: 厳格なポリシー（nonce ベース・推奨）**

```go
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func secureHandler(w http.ResponseWriter, r *http.Request) {
	nonce := generateNonce()

	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; "+
			"script-src 'nonce-"+nonce+"'; "+ // nonce 付きスクリプトのみ許可
			"style-src 'self'; "+
			"img-src 'self'; "+
			"connect-src 'self'; "+
			"frame-ancestors 'none'; "+
			"base-uri 'self'; "+
			"form-action 'self'")

	// HTML 内で nonce 属性を付与
	fmt.Fprintf(w, `
        <script nonce="%s">
            // このスクリプトだけが実行される
            // 攻撃者が注入した <script> タグには nonce がない → ブロック
            console.log('legitimate script');
        </script>
    `, nonce)
}
```

**nonce による防御の仕組み:**

```text
サーバーが生成: <script nonce="abc123">...</script>
CSP ヘッダー:  script-src 'nonce-abc123'

攻撃者が注入: <script>alert('XSS')</script>
               ↑ nonce 属性がない → ブラウザがブロック ✓

<script nonce="abc123">alert('XSS')</script>
               ↑ 攻撃者は nonce 値を知りえない → ブロック ✓
```

#### CSP が防ぐ攻撃と防げない攻撃

| 攻撃                         | CSP で防げる？                  | 理由                                        |
| :--------------------------- | :------------------------------ | :------------------------------------------ |
| インライン `<script>` XSS    | `script-src 'self'` で防ぐ      | インラインスクリプトがブロックされる        |
| 外部ドメインの悪意スクリプト | `script-src 'self'` で防ぐ      | 許可外ドメインがブロックされる              |
| `eval()` ベースの XSS        | `unsafe-eval` なしで防ぐ        | `eval()` がブロックされる                   |
| クリックジャッキング         | `frame-ancestors 'none'` で防ぐ | iframe 埋め込みがブロックされる             |
| CSRF                         | 防げない                        | CSRF はフォーム送信でありスクリプトではない |
| `<img onerror=...>` XSS      | `unsafe-inline` なしで防ぐ      | インラインイベントハンドラがブロックされる  |

#### 実装例: Go サーバーでの CSP ミドルウェア

```go
func cspMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := generateNonce()
		r.Header.Set("X-CSP-Nonce", nonce) // ハンドラーで参照可能に

		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; "+
				"script-src 'nonce-"+nonce+"'; "+
				"style-src 'self'; "+
				"img-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"report-uri /csp-report")

		next.ServeHTTP(w, r)
	})
}
```

#### CSP 違反の確認方法

```text
Chrome DevTools:
  Console タブ → "Refused to execute inline script because it violates
                   the following Content Security Policy directive: 'script-src'"

  Network タブ → "csp-report" への POST リクエストを確認

レポート内容の例:
{
  "csp-report": {
    "document-uri": "http://localhost:8080/xss/stored",
    "violated-directive": "script-src 'self'",
    "blocked-uri": "inline"
  }
}
```

### 発展的な攻撃手法

#### DOM-based XSS

従来の XSS とは異なり、サーバー側のレスポンスには脆弱性がなく、クライアント側の JavaScript で DOM 操作を行う際に発生します。

```javascript
// 脆弱なコード例
const hash = location.hash.substring(1); // URLの#以降を取得
document.getElementById("output").innerHTML = hash; // エスケープなしで挿入

// 攻撃URL:
// http://victim.com/#<img src=x onerror=alert('XSS')>
```

**対策:**

- `innerHTML` の代わりに `textContent` や `innerText` を使用
- DOMPurify 等のサニタイズライブラリ使用

#### Self-XSS (自己XSS)

攻撃者が「アクセス権限が必要」「特別なコマンド」と騙して、ユーザー自身に悪意のあるスクリプトを実行させます。

**対策:**

- ユーザー教育（不審なコードをコンソールに貼り付けない）
- 重要操作には再認証を要求
