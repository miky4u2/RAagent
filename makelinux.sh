#!/bin/bash
GOOS=linux GOARCH=amd64 go build -o ./runtime/bin/startagent  ./startagent/startagent.go
GOOS=linux GOARCH=amd64 go build -o ./runtime/bin/agent  ./agent/agent.go