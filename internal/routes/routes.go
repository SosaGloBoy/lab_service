package routes

import (
	"github.com/gin-gonic/gin"
	"lab/internal/handlers"
)

func SetupRoutes(router *gin.Engine, labHandler *handlers.LabHandler) {
	// Группа маршрутов для лаборатории
	labGroup := router.Group("/labs")
	{
		// Создание лаборатории
		labGroup.POST("", labHandler.CreateLabHandler)

		// Обновление лаборатории
		labGroup.PUT("/:id", labHandler.UpdateLabHandler)

		// Удаление лаборатории
		labGroup.POST("/:id/delete", labHandler.DeleteLabHandler)

		// Запуск лаборатории
		labGroup.POST("/:id/start", labHandler.StartLabHandler)

		// Остановка лаборатории
		labGroup.POST("/:id/stop", labHandler.StopLabHandler)

		// Выполнение команды в лаборатории
		labGroup.POST("/:id/execute-command", labHandler.ExecuteCommandHandler)

		labGroup.GET("/:id", labHandler.GetLabHandler)

		labGroup.POST("/:id/commit", labHandler.CommitLabHandler)

		labGroup.POST("/:id/deleteCommits", labHandler.DeleteCommitLabHandler)
	}
}
