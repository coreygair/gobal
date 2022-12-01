package main

import (
	"fmt"
	"go-balancer/internal/balancer"
	"go-balancer/internal/balancer/config"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func main() {
	// seed randomness
	rand.Seed(time.Now().UnixNano())

	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Missing expected argument: config-file\nExpected usage: balancer config-file")
		os.Exit(1)
	}
	if len(args) > 1 {
		fmt.Println("Extra arguments are being ignored...\nExpected usage: balancer config-file")
	}

	configPath := args[0]

	// pull config
	config, err := config.ReadConfig(configPath)
	if err != nil {
		fmt.Printf("Error opening config file: %s", err.Error())
		os.Exit(1)
	}

	b, err := balancer.NewBalancer(config)
	if err != nil {
		fmt.Printf("Error creating balancer: %s", err.Error())
		os.Exit(1)
	}

	modServer := balancer.NewModificationServer(&b)
	modServer.Start()

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: http.HandlerFunc(b.ServeHTTP),
	}

	// Listens on port for http connections
	// Creates new goroutine for each connection,
	// 	which then calls s.Handler to handle the request
	s.ListenAndServe()
}
