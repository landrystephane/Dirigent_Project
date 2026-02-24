package main

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info(r.URL.Path)
	})

	addr := "127.0.0.1:6000"
	fmt.Println("Listening on", addr)
	http.ListenAndServe(addr, nil)
}
