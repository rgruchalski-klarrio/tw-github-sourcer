package main

import (
	"io"
	"net/http"
)

func getRoot(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

func main() {
	http.HandleFunc("/", getRoot)
	http.ListenAndServe(":8080", nil)
}
