package wsproxy_test

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gopyai/go-wsproxy"
)

var (
	isTLS   = false
	cerFile = "my.cer"
	keyFile = "my.key"

	port        = 8001
	timeoutSecs = 30

	config = map[string]string{
		"/ping/": "localhost:8081",
		"/chat/": "localhost:8082",
	}
)

func Example() {
	mux := http.NewServeMux()

	// Initialize mux with path based websocket config
	for p, v := range config {
		mux.Handle(p, wsproxy.Handler(p, v))
	}

	// Start HTTP server
	log.Println("Starting on port", port)

	timeout := time.Duration(timeoutSecs) * time.Second
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}

	var e error
	if isTLS {
		e = srv.ListenAndServeTLS(cerFile, keyFile)
	} else {
		e = srv.ListenAndServe()
	}
	if e != nil {
		fmt.Println("Error:", e)
	}
}
