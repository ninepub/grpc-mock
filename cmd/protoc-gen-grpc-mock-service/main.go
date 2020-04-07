package main

import (
	"github.com/ninepub/grpc-mock/internal/proto"
	"github.com/ninepub/grpc-mock/internal/types"
)

func main() {
	proto.GenerateGrpcCode(types.ServiceTemplate)
}
