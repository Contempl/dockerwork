package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        name, _ := os.Hostname()
        fmt.Fprintf(w, "Сервер: %s, версия: 1.0.0\n", name)
    })

    http.ListenAndServe(":8080", nil)
}