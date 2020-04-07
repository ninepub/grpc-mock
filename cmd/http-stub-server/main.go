package main

import (
	"context"
	"flag"
	"github.com/ninepub/grpc-mock/pkg/stubsrv"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func run() {
	addr := flag.String("addr", "", "Adress the admin service will bind to. Default to localhost, set to 0.0.0.0 to use from another machine")
	port := flag.Int("port", 4770, "Port of stub admin service")
	stubPath := flag.String("stub", "", "Path where the stub files are (Optional)")

	flag.Parse()
	// run admin stub server

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go stubsrv.StartStubServer(ctx, &stubsrv.Server{Addr: *addr, Port: *port, StubPath: *stubPath})

	<-done
	log.Println("Exiting HTTP stub server...")
	return
}

func main() {
	run()
}
