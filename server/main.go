package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const namespace = "ndn"

func main() {
	listenAddress := flag.String("listen", ":9090", "gRPC listen address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	routerMap, err := startRouterWatcher(ctx, namespace)
	if err != nil {
		log.Fatalf("start Router watcher: %v", err)
	}

	server, listener, err := newGRPCServer(*listenAddress, routerMap)
	if err != nil {
		log.Fatalf("start gRPC server: %v", err)
	}
	defer listener.Close()
	go server.Serve(listener)
	log.Printf("NDN DHCP server listening on %s", *listenAddress)
	<-ctx.Done()
	server.GracefulStop()
}
