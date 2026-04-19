package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

// In-memory data for demo
type Data struct {
	sync.Mutex
	Comments []string
	Email    string
}

var data = &Data{
	Comments: []string{"Welcome to the workshop!"},
	Email:    "victim@example.com",
}

func main() {
	mux := http.NewServeMux()

	// Victim Site
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/xss/stored", storedXSSHandler)
	mux.HandleFunc("/xss/reflected", reflectedXSSHandler)
	mux.HandleFunc("/update-email", updateEmailHandler)
	mux.HandleFunc("/transfer", transferHandler)

	log.Println("Victim site starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<h1>Victim Site</h1>
		<p>Current email: <strong>%s</strong></p>
		<ul>
			<li><button onclick="alert(document.cookie)">Show Cookie</button></li>
			<li><button onclick="document.cookie='session_id=secret-session-123; path=/'; alert('Logged in! Cookie has been set.')">Login (set session cookie)</button></li>
			<li><button onclick="document.cookie='session_id=; path=/; max-age=0'; alert('Logged out. Cookie has been cleared.')">Logoff (clear session cookie)</button></li>
			<li><a href="/xss/stored">Stored XSS Demo</a></li>
			<li><a href="/xss/reflected?q=test">Reflected XSS Demo</a></li>
			<li><a href="/transfer">Transfer Page (Clickjacking Demo)</a></li>
		</ul>
		<hr>
		<h2>Links to Attacker Site (for exercises)</h2>
		<ul>
			<li><a href="http://localhost:8081/csrf">CSRF Attack Page (port 8081)</a></li>
			<li><a href="http://localhost:8081/clickjacking">Clickjacking Attack Page (port 8081)</a></li>
		</ul>
	`, data.Email)
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
	fmt.Fprint(w, "<h1>Stored XSS</h1>")
	fmt.Fprint(w, "<p>Submitted content is stored and displayed as-is.</p>")
	fmt.Fprint(w, "<form method='POST'><input name='comment' style='width:300px'><input type='submit' value='Submit'></form>")
	fmt.Fprint(w, "<h2>Comments</h2><ul>")
	for _, c := range data.Comments {
		// Vulnerability: user input is rendered unescaped
		fmt.Fprintf(w, "<li>%s</li>", c)
	}
	fmt.Fprint(w, "</ul><a href='/'>Back to home</a>")
}

func reflectedXSSHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Vulnerability: query parameter is embedded directly into HTML
	fmt.Fprintf(w, "<h1>Search Results</h1><p>0 results found for \"%s\".</p><a href='/'>Back to home</a>", q)
}

func updateEmailHandler(w http.ResponseWriter, r *http.Request) {
	// Simplified "authentication" check
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
		fmt.Fprintf(w, "Email updated to <strong>%s</strong>. <a href='/'>Back to home</a>", newEmail)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<h1>Update Email</h1>
		<form method="POST">
			New email: <input name="email" value="%s">
			<input type="submit" value="Update">
		</form>
	`, data.Email)
}

func transferHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h2>Transfer of $1000 completed!</h2><a href='/'>Back to home</a>")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Vulnerability: no X-Frame-Options header, allowing iframe embedding from other sites
	fmt.Fprintf(w, `
		<style>
			body { font-family: sans-serif; text-align: center; padding-top: 50px; }
			.btn { background: #ff4444; color: white; padding: 20px 40px; border: none; font-size: 20px; cursor: pointer; border-radius: 5px; }
		</style>
		<h1>Transfer Confirmation</h1>
		<p>Send <strong>$1000</strong> to "Malicious Hacker"?</p>
		<form method="POST">
			<button class="btn">Confirm Transfer</button>
		</form>
	`)
}
