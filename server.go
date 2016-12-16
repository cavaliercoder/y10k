package main

import (
	"net/http"
	"time"
)

func serve(path, addr string) {
	fs := http.FileServer(http.Dir(path))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		defer func() {
			d := time.Since(t)
			Printf("%v %v %v %v\n", r.RemoteAddr, r.Method, r.URL.Path, d)
		}()

		fs.ServeHTTP(w, r)
	})

	Printf("Listening on %v\n", addr)
	http.ListenAndServe(addr, handler)
}
