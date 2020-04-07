package main

import (
	"flag"

	"github.com/ninepub/grpc-mock/internal/server"
	"github.com/ninepub/grpc-mock/internal/types"
)

func run() {
	grpcPort := flag.String("grpc-port", "50051", "Port of gRPC tcp service")
	grpcHost := flag.String("grpc-host", "", "Address the gRPC service will bind to. Default to localhost, set to 0.0.0.0 to use from another machine")
	stubHost := flag.String("stub-host", "127.0.0.1", "Host of stub service")
	stubPort := flag.String("stub-port", "4770", "Port of stub service")
	output := flag.String("out-path", "", "The path where server file needs to be generated")
	pkgFile := flag.String("pkg-def-path", "./package", "The path where package definitions are kept")
	pkgSuffix := flag.String("pkg-suffix", "", "The package suffix")

	flag.Parse()

	// generate pb.go and server server based on proto
	param := &types.Server{
		GrpcHost:      *grpcHost,
		GrpcPort:      *grpcPort,
		Output:        *output,
		PackageSuffix: *pkgSuffix,
		StubHost:      *stubHost,
		StubPort:      *stubPort,
	}

	param.Packages = server.GeneratePackageDef(*pkgFile)
	server.GenerateServiceRegister(param, types.ServerTemplate)
	server.GenerateStub(param, types.StubTemplate)
}

func main() {
	run()
}
