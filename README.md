# SCUA-API

## Docker

### Teste inicial

### Comandos:
Criar projeto:
go mod init github.com/heronhurpia/scua-api

Criar arquivo main.go:
package main

import "fmt"

func main() {
	fmt.Println("Hello, world.")
}

go install github.com/heronhurpia/scua-api

docker build -t scua-api .
docker run -it --rm --name my-running-app scua-api
docker run --rm -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.21 go build -v

