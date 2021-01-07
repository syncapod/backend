#!/bin/sh
# compile protocol buffers

protoc -I=C:/users/sam/projects/protos/syncapod-protos/ \
	--go_out=internal/protos/ \
	--go-grpc_out=internal/protos/ \
	C:/users/sam/projects/protos/syncapod-protos/*