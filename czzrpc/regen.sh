#!/bin/sh

protoc -I. czzrpc.proto --go_out=plugins=grpc:pb