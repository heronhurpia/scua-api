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

Comando de teste:
curl -H "Token: 744qy4iapitwh3q6" 'http://localhost:3011/api/get_scua_list?limit=10&offset=10'

### todo
+ ~~Separar função para busca na url~~
+ ~~Simular resposta errada da api~~
+ ~~Criar validador de resposta com lista de receptores~~
+ ~~criar lista de constantes no controller~~
+ ~~Nome do arquivo~~
+ ~~quantidade de busca~~
+ ~~url da API~~
+ ~~Verificar se arquivo existe na inicializaçao, se não existe, criar~~