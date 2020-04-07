#!/bin/bash
echo "Building grpc service generator"
go install ./cmd/protoc-gen-grpc-mock-service

echo "Building grpc server generator"
go install ./cmd/grpc-mock-server
