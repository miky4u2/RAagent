#!/bin/bash
#GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui -o ./runtime/bin/startagent.exe  ./startagent/startagent.go
#GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui -o ./runtime/bin/agent.exe  ./agent/agent.go
GOOS=windows GOARCH=amd64 go build  -o ./runtime/bin/startagent.exe  ./startagent/startagent.go
GOOS=windows GOARCH=amd64 go build  -o ./runtime/bin/agent.exe  ./agent/agent.go
