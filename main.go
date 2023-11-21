package main

import (
	"github.com/gin-gonic/gin"
	"github.com/heronhurpia/scua-api/controllers"
)

func main() {

	//	controllers.InitDB()
	controllers.Init()

	r := gin.Default()
	r.Static("/assets", "./assets")

	/* Páginas */
	views := r.Group("/")
	views.GET("/find-scua/:scua", controllers.FindScua)

	r.Run(":8080")
}
