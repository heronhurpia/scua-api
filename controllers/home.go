package controllers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
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
var m sync.Mutex

type response struct {
	Result  bool   `json:"res"`
	Message string `json:"message"`
}

// SafeCounter is safe to use concurrently.
type SafeSet struct {
	mu   sync.Mutex
	scua set.Set
}

// getAlbums responds with the list of all albums as JSON.
func FindScua(c *gin.Context) {

	var r bool
	var m string

	// scua que deve ser procurado
	scua := c.Param("scua")
	//log.Println([]byte(scua))

	if scua_set.Has(scua) {
		r = true
		m = "Encontrado " + scua
	} else {
		r = false
		m = fmt.Sprintf("%T %q não localizado em %d itens", scua, scua, scua_set.Len())
	}

	// Resposta da solicitação
	var res = []response{
		{
			Result:  r,
			Message: m},
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

	fmt.Println("Lista de receptores:")
	r := bufio.NewReader(f)
	s, _, e := r.ReadLine()
	for e == nil {
		scua_set.Insert(string(s))
		s, _, e = r.ReadLine()
	}

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

	time.Sleep(5 * time.Second)

	for {
		// Início da verificação
		offset := scua_set.Len()
		fmt.Printf("Total de receptores na memória: %d\n", offset)

		// Create an HTTP client
		client := &http.Client{}

		// Fazer a busca de novos receptores
		var url string = fmt.Sprintf("%s?limit=%d&offset=%d", api_url, api_limit, offset)

		// Create an HTTP request with custom headers
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Error creating HTTP request:", err)
			return
		}
		req.Header.Add(api_authorization, api_token)

		// Send the HTTP request
		fmt.Printf("req: %s\n", req.URL)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending HTTP request:", err)
			return
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading HTTP response body:", err)
			return
		}

		// Print the response body
		fmt.Println("Resposta:")
		fmt.Println(string(body))

		// Separa resposta linha por linha e salva na lista de scua
		sc := bufio.NewScanner(strings.NewReader(string(body)))
		for sc.Scan() {
			// Salva dados garantindo não concorrente
			m.Lock()
			scua_set.Insert(string(sc.Text()))
			m.Unlock()
		}

		// Acrescenta os dados recebidos ao arquivo
		f, err := os.OpenFile(data_filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}
		if _, err = f.WriteString(string(body)); err != nil {
			panic(err)
		}
		f.Close()

		/* Tempo para nova varredura  */
		time.Sleep(10 * time.Minute)
	}
}
