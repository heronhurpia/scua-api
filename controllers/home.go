package controllers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-collections/collections/set"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Lista de constantes
var scua_set *set.Set
var data_filename = "scua_data.idx"

// "H 'Token: 744qy4iapitwh3q6' 'http://localhost:3011/api/get_scua_list?limit=1000&offset=%d'", offset
var api_url = "http://localhost:3011/api/get_scua_list"
var api_limit = 10
var api_authorization = "Token"
var api_token = "744qy4iapitwh3q6"
var scua_lock sync.RWMutex

type response struct {
	Result  bool   `json:"res"`
	Message string `json:"message"`
}

// getAlbums responds with the list of all albums as JSON.
func FindScua(c *gin.Context) {

	var r bool
	var msg string

	// scua que deve ser procurado
	scua := c.Param("scua")
	//log.Println([]byte(scua))

	scua_lock.RLock()
	has_scua := scua_set.Has(scua)
	len_scua := scua_set.Len()
	scua_lock.RUnlock()

	if has_scua {
		r = true
		msg = "Encontrado " + scua
	} else {
		r = false
		msg = fmt.Sprintf("%T %q não localizado em %d itens", scua, scua, len_scua)
	}

	// Resposta da solicitação
	var res = []response{
		{
			Result:  r,
			Message: msg},
	}

	c.IndentedJSON(http.StatusOK, res)
}

// Abre arquivo lista de scua e salva na RAM
func Init() {

	// Verifica se arquivo existe e se não existe, criar
	if _, err := os.Stat(data_filename); os.IsNotExist(err) {
		_, err := os.Create(data_filename)
		if err != nil {
			panic(err)
		}
	}

	// Abre arquivo com a lista de receptores
	f, _ := os.Open(data_filename)
	defer f.Close()

	// Inicia variável com lista de receptores em RAM
	scua_set = set.New()

	r := bufio.NewReader(f)
	s, _, e := r.ReadLine()
	for e == nil {
		scua_set.Insert(string(s))
		s, _, e = r.ReadLine()
	}

	// Loop de atualização da lista de receptores
	go updateScuaList()
}

// Cosulat DB e cria arquivo
func InitDB() uint64 {
	url := "postgresql://m2/logprodv6"
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect: %q %v", url, err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), `select data_value 
	from tb_items_metadata m 
	join tb_products_metadata p on m.metadata_id = p.metadata_id 
	join tb_items i on i.item_id = m.item_id 
	join vw_products r on r.product_id = i.product_id 
	where p.metadata_code in('vUA', 'SCUA', 'NSC')`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to exec sql: %v", err)
		os.Exit(2)
	}

	f, _ := os.OpenFile(data_filename, os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()

	var amount uint64
	for rows.Next() {
		var scua string
		if err := rows.Scan(&scua); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to read row: %v", err)
			os.Exit(3)
		}
		//fmt.Fprintf(os.Stdout, "Added: %q\n", scua)
		f.Write([]byte(scua))
		f.Write([]byte("\n"))
		amount += 1
	}

	return amount
}

func Update(c *gin.Context) {

	// Atualiza DB e retorna total de registros
	amount := InitDB()

	// Resposta da solicitação
	var res = []response{
		{
			Result:  true,
			Message: "Banco dados atualizado"},
		{
			Result:  true,
			Message: fmt.Sprintf("Total de registros: %d", amount)},
	}

	c.IndentedJSON(http.StatusOK, res)
}

// Roda periódicamente buscando atualizações no banco de dados
func updateScuaList() {

	// Atraso inicial apenas para não misturar mensagens de log
	time.Sleep(2 * time.Second)

	for {
		fmt.Printf("\n\n\n")
		// scua_lock.RLock()
		// scua_set.Do(func(i interface{}) {
		// 	fmt.Printf("%s\n", i)
		// })
		offset := scua_set.Len()
		// scua_lock.RUnlock()

		// // Busca lista de scua
		fmt.Println(time.Now().Format("15:04:05"), " - total de receptores na memória: ", offset)
		body, err := get_scua_list(offset)

		// // Se não consegiu contato com API, entrar em standby
		if err != nil {
			fmt.Println("Erro de conexão com API")
			time.Sleep(60 * time.Minute)
			continue
		}

		// // Caso não existam novas linhas, entrar em standby por 1 hora
		if len(body) == 0 {
			fmt.Println("Não foram encontrados novos scuas no db")
			time.Sleep(60 * time.Minute)
			continue
		}

		// // Cria variável
		var scua_valid *set.Set = set.New()

		// Separa resposta linha por linha e salva na lista de scua
		var lines int
		scua_api := bufio.NewScanner(strings.NewReader(string(body)))
		for scua_api.Scan() {
			// Verifica se a linha corresponde a um receptor válido
			if isValidScua(scua_api.Text()) {
				scua_valid.Insert(string(scua_api.Text()))
				lines += 1

			} else {
				fmt.Printf("%q inválido\n", scua_api.Text())
			}
		}
		fmt.Println(time.Now().Format("15:04:05"), " - linhas recebidas via api: ", lines)

		// Número de receptores recebidos via API
		final := scua_valid.Len()
		fmt.Println(time.Now().Format("15:04:05"), " - total de receptores recebidos via api: ", final)
		// scua_valid.Do(func(i interface{}) {
		// 	fmt.Printf("%s\n", i)
		// })

		// Filtra apenas os receptores que ainda não estão na lista
		var scua_new *set.Set = set.New()

		scua_lock.RLock()
		scua_new = scua_valid.Difference(scua_set)
		scua_lock.RUnlock()

		// new := scua_new.Len()
		// fmt.Println(time.Now().Format("15:04:05"), " - novos receptores recebidos via api: ", new)
		// scua_new.Do(func(i interface{}) {
		// 	fmt.Printf("%s\n", i)
		// })

		/* Se não houve alteração na quantidade receptores, entra em standby */
		if scua_new.Len() == 0 {
			fmt.Println(time.Now().Format("15:04:05"), " - não houve alteração no total de receptores")
			time.Sleep(time.Minute)
			continue
		}

		// Abre arquivo destino dos dados
		f, err := os.OpenFile(data_filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			fmt.Println(time.Now().Format("15:04:05"), " - falha ao abrir arquivo")
			time.Sleep(10 * time.Minute)
			continue
		}

		// Salva dados no arquivo
		scua_new.Do(func(i interface{}) {
			var data string = fmt.Sprintf("%s\n", i)
			if _, err = f.WriteString(string(data)); err != nil {
				fmt.Println(time.Now().Format("15:04:05"), " - falha ao salvar dados")
			}
		})
		f.Close()

		// Atualiza variável com lista de scua
		scua_lock.Lock()
		scua_new.Do(func(i interface{}) {
			scua_set.Insert(i)
		})
		total := scua_set.Len()
		scua_lock.Unlock()
		fmt.Println(time.Now().Format("15:04:05"), " - total de receptores após união: ", total)

		fmt.Println(time.Now().Format("15:04:05"), " - fim da atualização da lista de scuas")
		time.Sleep(time.Minute)
	}
}

// Verifica se scua é válido
func isValidScua(scua string) bool {
	return isNagra(scua) || isVerimatrix(scua)
}

func isNagra(scua string) bool {

	result := true

	// Cada scua tem o tamanho fixo de 12 bytes
	if len(scua) != 12 {
		result = false
	} else {
		// Os 12 carateres tem que ser numéricos
		if _, err := strconv.Atoi(scua); err != nil {
			result = false
		}
	}

	// if result {
	// 	fmt.Printf("%q: Nagra\n", scua)
	// }
	return result
}

func isVerimatrix(scua string) bool {

	result := true

	// Cada scua tem o tamanho fixo de 12 bytes
	if len(scua) != 12 {
		result = false
	} else {
		// O primeiro caracter tem que er "N"
		if scua[:1] != "N" {
			result = false
		}

		// Os 11 últimos carateres tem que ser numéricos
		if _, err := strconv.Atoi(scua[1:]); err != nil {
			result = false
		}
	}

	// if result {
	// 	fmt.Printf("%q: Verimatrix\n", scua)
	// }
	return result
}

// Busca lista de recetores na api
func get_scua_list(offset int) (string, error) {

	// Create an HTTP client
	client := &http.Client{}

	// Fazer a busca de novos receptores
	var url string = fmt.Sprintf("%s?limit=%d&offset=%d", api_url, api_limit, offset)

	// Create an HTTP request with custom headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return "", err
	}
	req.Header.Add(api_authorization, api_token)

	// Send the HTTP request
	fmt.Println(time.Now().Format("15:04:05"), " -", req.URL)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return "", err
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading HTTP response body:", err)
		return "", err
	}

	// Verifica resposta da api
	var lines int = strings.Count(string(body), "\n")
	fmt.Println(time.Now().Format("15:04:05"), " - total de linhas recebidas da API: ", lines)

	return string(body), nil

}
