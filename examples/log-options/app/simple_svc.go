package main

import "net/http"

func main() {
	go http.ListenAndServe(":80", http.HandlerFunc(handleQuery))
	http.ListenAndServe(":8080", http.HandlerFunc(handleHealthCheck))
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	w.WriteHeader(200)
	_, _ = w.Write([]byte(path))
}
