#!/bin/bash

prototool generate proto

grpc-mock-server --out-path ./generated/grpc --pkg-suffix github.com/ninepub/grpc-mock/generated/grpc/