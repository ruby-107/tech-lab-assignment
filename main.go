package main

import (
	"employee-file-upload/controllers"
	"employee-file-upload/database"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Connect to MySQL
	database.ConnectDB()

	//Connect to redis
	database.ConnectRedis()

	// Routes
	r.POST("/upload", controllers.ImportExcel)
	r.GET("/employees", controllers.GetEmployees)
	r.POST("/employees", controllers.CreateEmployee)
	r.PUT("/employees/:id", controllers.UpdateEmployee)
	r.DELETE("/employees/:id", controllers.DeleteEmployee)
	r.GET("/employees/:id", controllers.GetEmployeeByID)

	r.Run(":8080") // Start server
}
