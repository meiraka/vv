package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

// App serves http request.
func App(config ServerConfig) {
	http.HandleFunc("/", handler)
	http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil)
}
