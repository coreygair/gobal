package main

import (
	"net/http"
	"os"
	"strconv"
)

func main() {
	args := os.Args[1:]

	if len(args) < 2 {
		os.Exit(1)
	}

	// check port no. is present
	port, err := strconv.Atoi(args[0])
	if err != nil || port < 0 || port > 65535 {
		os.Exit(2)
	}

	message := args[1]

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(message))
	}

	s := &http.Server{
		Addr:    ":" + args[0],
		Handler: http.HandlerFunc(handler),
	}
	s.ListenAndServe()
}
