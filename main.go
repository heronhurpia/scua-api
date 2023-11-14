package main

import (
	"github.com/gin-gonic/gin"
	"github.com/heronhurpia/scua-api/controllers"
	"github.com/heronhurpia/scua-api/middlewares"
	"github.com/heronhurpia/scua-api/models"
)

func main() {

	//	controllers.InitDB()
	controllers.Init()

	models.ConnectDataBase()

	r := gin.Default()
	r.Static("/assets", "./assets")
	r.LoadHTMLGlob("templates/**/*")

	/* Lista de rotas */
	public := r.Group("/api")
	public.POST("/register", controllers.Register)
	public.POST("/login", controllers.Login)

	/* Páginas */
	views := r.Group("/")
	views.GET("/find-scua/:scua", controllers.FindScua)
	views.GET("/find-scua", controllers.FindScua)
	views.GET("/update", controllers.Update)

	protected := r.Group("/api/admin")
	protected.Use(middlewares.JwtAuthMiddleware())
	protected.GET("/user", controllers.CurrentUser)

	r.Run(":8080")
}
