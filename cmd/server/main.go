package main

import (
	"github.com/gin-gonic/gin"
	"lab/internal/config"
	"lab/internal/handlers"
	"lab/internal/repository"
	"lab/internal/routes"
	"lab/internal/service"
	"log"
	"log/slog"
	"os"
)

func main() {
	cfg := config.LoadConfig()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
	}))

	db, err := config.InitDB(cfg)
	if err != nil {
		logger.Error("Error initializing database", "error", err)
		return
	}

	labRepository := repository.NewLabRepository(db, logger)

	labService := service.NewLabService(labRepository, cfg.TaskServiceURL, logger)

	labHandler := handlers.NewLabHandler(labService, cfg.TaskServiceURL, logger)

	router := gin.Default()
	routes.SetupRoutes(router, labHandler)

	log.Printf("Server running on port %s", cfg.ServerPort)
	if err := router.Run(cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}
