package main

import (
	"log"
	"os"
	"path/filepath"

	"github-repo-manager/handlers"
	"github-repo-manager/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := models.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := os.Chmod(filepath.Join(".", "scripts", "create_repo.sh"), 0755); err != nil {
		log.Printf("Warning: chmod script failed: %v", err)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	handler := handlers.NewRepoHandler()

	api := r.Group("/api")
	{
		api.GET("/repos", handler.ListRepos)
		api.GET("/repos/:id", handler.GetRepo)
		api.POST("/repos/batch", handler.BatchCreate)
		api.POST("/repos/:id/retry", handler.RetryRepo)
		api.DELETE("/repos/:id", handler.DeleteRepo)
		api.DELETE("/repos", handler.DeleteAll)
		api.GET("/stats", handler.Stats)
	}

	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/", "./frontend/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
