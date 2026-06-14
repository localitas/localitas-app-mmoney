package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/localitas/localitas-app-mmoney"
	"github.com/localitas/localitas-go"
)

var (
	version = "dev"
	commit  = "unknown"
)

func envOrFileToken() string {
	if t := os.Getenv("LOCALITAS_API_TOKEN"); t != "" {
		return t
	}
	return client.DefaultToken()
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("mmoney-server %s (commit: %s)\n", version, commit)
		os.Exit(0)
	}

	var (
		listen   = flag.String("listen", ":0", "listen address")
		coreURL  = flag.String("core-url", client.DefaultCoreURL(), "base URL of the Localitas core API")
		basePath = flag.String("base-path", "/", "URL prefix for <base href>")
		token    = flag.String("token", envOrFileToken(), "bearer token")
	)
	flag.Parse()

	ctx := context.Background()
	c := client.New(*coreURL)
	if *token != "" {
		c = c.WithToken(*token)
	}

	app := mmoney.New(c, *basePath)

	dbID, err := app.Install(ctx)
	if err != nil {
		log.Fatalf("install: %v", err)
	}
	log.Printf("MMoney database ready: %s", dbID)

	if err := app.InitStore(*coreURL, dbID, *token); err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer app.Store.Close()

	app.SetCoreAccess(*coreURL, *token)

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	mux.HandleFunc("GET /health.json", mmoney.HandleHealth)

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	appURL := fmt.Sprintf("http://localhost:%d", addr.Port)
	fmt.Printf("mmoney-server listening on %s\n", appURL)

	if err := c.RegisterService(ctx, "mmoney", appURL); err != nil {
		log.Printf("service registry failed: %v", err)
	}

	mmoney.RegisterSyncAutomation(*coreURL, *token, appURL)

	shutdown, err := mmoney.BroadcastMDNS(addr.Port, mmoney.DefaultHealth.Name)
	if err != nil {
		log.Printf("mDNS broadcast failed: %v", err)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		if shutdown != nil {
			shutdown()
		}
		os.Exit(0)
	}()

	if err := http.Serve(ln, mux); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
