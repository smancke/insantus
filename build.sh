#!/bin/bash -x

go get ./...
go build -ldflags '-linkmode external -extldflags -static -w' -o insantus github.com/smancke/insantus
docker build -t smancke/insantus .

