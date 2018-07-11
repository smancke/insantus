#!/bin/bash -x

go get ./...
go build -ldflags '-linkmode external -extldflags -static -w' -o detailedDataSource .
docker build -t detaileddatasource:example .
docker stack deploy -c docker-compose.yml example