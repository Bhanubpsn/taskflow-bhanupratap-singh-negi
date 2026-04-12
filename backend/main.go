package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/controllers"
	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/database"
	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/middleware"
	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	// DB connection
	pool, err := database.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	if err := database.RunMigrations(os.Getenv("DATABASE_URL"), "./migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	controllers.DB = pool

	router := gin.New()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf(
			"%s - [%s] \"%s %s %d %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	}))
	router.Use(gin.Recovery())
	routes.UserRoutes(router)

	protected := router.Group("/")
	protected.Use(middleware.RequireAuth())
	{
		routes.ProjectRoutes(protected)
		routes.TaskRoutes(protected)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(router.Run(":" + port))
}
