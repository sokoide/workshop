package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/sleep/", sleepHandler)
	mux.HandleFunc("/error/", errorHandler)

	srv := &http.Server{
		Addr:              ":80",
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("backend ok\n"))
}

func sleepHandler(w http.ResponseWriter, r *http.Request) {
	sec, ok := pathInt(r.URL.Path, "/sleep/")
	if !ok {
		http.NotFound(w, r)
		return
	}

	time.Sleep(time.Duration(sec) * time.Second)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("slept %d seconds\n", sec)))
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt(r.URL.Path, "/error/")
	if !ok || code < 100 || code > 599 {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(code)
	_, _ = w.Write([]byte(fmt.Sprintf("error %d\n", code)))
}

func pathInt(path, prefix string) (int, bool) {
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	v := strings.TrimPrefix(path, prefix)
	if v == "" || strings.Contains(v, "/") {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}
