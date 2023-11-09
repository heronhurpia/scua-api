package controllers

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/golang-collections/collections/set"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

var scua_set *set.Set
var data_filename = "scua_data.idx"

type response struct {
	Result  bool   `json:"res"`
	Message string `json:"message"`
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

func Init() {
	f, _ := os.Open(data_filename)
	defer f.Close()

	scua_set = set.New()

	r := bufio.NewReader(f)
	s, _, e := r.ReadLine()
	for e == nil {
		//fmt.Println(string(s))
		scua_set.Insert(string(s))
		s, _, e = r.ReadLine()
	}
}

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
