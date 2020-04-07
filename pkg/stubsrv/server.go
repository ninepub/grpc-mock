package stubsrv

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/ninepub/grpc-mock/internal/stub"
)

type Server struct {
	Addr     string
	Port     int
	StubPath string
}

func StartStubServer(ctx context.Context, params *Server) {
	addr := params.Addr + ":" + strconv.Itoa(params.Port)
	router := stub.CreateRouter(params.StubPath)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Println("HTTP Stub Server :: Started : ", addr)

	defer func() {
		log.Println("HTTP Stub Server :: Shutdown command is issued..")
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("HTTP Stub server :: Shutdown Failed:%+v", err)
		}
		log.Println("HTTP Stub Server :: Exited Properly..")
	}()

	<-ctx.Done()
	return
}
