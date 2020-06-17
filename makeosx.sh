#!/bin/bash
GOOS=darwin GOARCH=amd64 go build -o ./runtime/bin/startagent  ./startagent/startagent.go
GOOS=darwin GOARCH=amd64 go build -o ./runtime/bin/agent  ./agent/agent.go