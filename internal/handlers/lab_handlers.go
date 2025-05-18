package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"lab/internal/model"
	"lab/internal/service"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type LabHandler struct {
	LabService     *service.LabService
	TaskServiceURL string // URL для доступа к сервису заданий
	Logger         *slog.Logger
}

// Конструктор для LabHandler
func NewLabHandler(labService *service.LabService, taskServiceURL string, logger *slog.Logger) *LabHandler {
	return &LabHandler{
		LabService:     labService,
		TaskServiceURL: taskServiceURL,
		Logger:         logger,
	}
}

// Метод для проверки существования задания и получения Docker-образа
func (h *LabHandler) checkTaskExists(taskID uint) (string, error) {
	// Запрос к сервису заданий для получения Docker-образа для задания
	url := fmt.Sprintf("%s/tasks/%d", h.TaskServiceURL, taskID)
	resp, err := http.Get(url)
	if err != nil {
		h.Logger.Error("Error checking task existence", "error", err)
		return "", fmt.Errorf("error checking task existence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("task not found")
	}

	// Извлекаем путь к Docker-образу
	var response struct {
		Task struct {
			VMImagePath string `json:"vm_image_path"`
		} `json:"task"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		h.Logger.Error("Error unmarshaling task response", "error", err)
		return "", fmt.Errorf("error unmarshaling task response: %w", err)
	}

	return response.Task.VMImagePath, nil
}

func (h *LabHandler) CreateLabHandler(c *gin.Context) {
	if c.Request.ContentLength == 0 {
		h.Logger.ErrorContext(c, "Empty request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request body is empty"})
		return
	}
	var request struct {
		TaskID      uint   `json:"task_id" binding:"required"`
		VMImagePath string `json:"vm_image_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.Logger.ErrorContext(c, "Failed to bind request data", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON: expected {task_id: number, vm_image_path: string}",
		})
		return
	}
	h.Logger.InfoContext(c, "Creating lab",
		"task_id", request.TaskID,
		"vm_image_path", request.VMImagePath,
	)
	ctx := c.Request.Context()
	containerID, accessURL, labID, err := h.LabService.CreateLab(ctx, request.TaskID, request.VMImagePath)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to create lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start lab container",
			"details": err.Error(), // Можно убрать в production
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"container_id": containerID,
		"access_url":   accessURL,
		"lab_id":       labID,
	})
}
func (h *LabHandler) UpdateLabHandler(c *gin.Context) {
	var lab model.Lab
	if err := c.ShouldBindJSON(&lab); err != nil {
		h.Logger.ErrorContext(c, "Failed to bind lab data", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Lab Data"})
		return
	}

	ctx := c.Request.Context()
	err := h.LabService.UpdateLab(ctx, &lab)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to update lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update lab"})
		return
	}

	h.Logger.InfoContext(c, "Lab updated", "lab", lab)
	c.JSON(http.StatusOK, gin.H{"message": "Lab updated successfully"})
}

// Обработчик для удаления лаборатории
func (h *LabHandler) DeleteLabHandler(c *gin.Context) {
	labIDParam := c.Param("id")
	labID, err := strconv.Atoi(labIDParam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
		return
	}

	ctx := c.Request.Context()
	err = h.LabService.DeleteLab(ctx, labID)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to delete lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete lab"})
		return
	}

	h.Logger.InfoContext(c, "Lab deleted", "lab_id", labID)
	c.JSON(http.StatusOK, gin.H{"message": "Laboratory deleted successfully"})
}

// Обработчик для получения лаборатории по ID
func (h *LabHandler) GetLabHandler(c *gin.Context) {
	labIDParam := c.Param("id")
	labID, err := strconv.Atoi(labIDParam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
		return
	}

	ctx := c.Request.Context()
	lab, err := h.LabService.GetLab(ctx, uint(labID))
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to get lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get lab"})
		return
	}

	h.Logger.InfoContext(c, "Lab found", "lab", lab)
	c.JSON(http.StatusOK, gin.H{"lab": lab})
}

// Обработчик для запуска лаборатории (контейнера)
func (h *LabHandler) StartLabHandler(c *gin.Context) {
	labIDParam := c.Param("id")
	labID, err := strconv.Atoi(labIDParam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
		return
	}

	// Получаем Docker-образ через API
	vmImagePath, err := h.checkTaskExists(uint(labID))
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to get task Docker image", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	ctx := c.Request.Context()
	containerID, err := h.LabService.StartLab(ctx, &model.Lab{ID: uint(labID)}, vmImagePath)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to start container", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start container"})
		return
	}

	h.Logger.InfoContext(c, "Lab started successfully", "container_id", containerID)
	c.JSON(http.StatusOK, gin.H{"message": "Laboratory started successfully", "container_id": containerID})
}

// Обработчик для остановки лаборатории
func (h *LabHandler) StopLabHandler(c *gin.Context) {
	labIDParam := c.Param("id")
	labID, err := strconv.Atoi(labIDParam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
		return
	}

	ctx := c.Request.Context()
	err = h.LabService.StopLab(ctx, labID)
	if err != nil {
		h.Logger.ErrorContext(c, "Error stopping lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not stop lab"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lab stopped"})
}

func (h *LabHandler) ExecuteCommandHandler(c *gin.Context) {
	var request struct {
		ContainerID string `json:"container_id"`
		Command     string `json:"command"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.Logger.ErrorContext(c, "Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if request.ContainerID == "" || request.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container_id and command are required"})
		return
	}

	commandArgs := strings.Fields(request.Command)

	output, _ := h.LabService.ExecuteCommand(c.Request.Context(), request.ContainerID, commandArgs)

	c.JSON(http.StatusOK, gin.H{"output": strings.TrimSpace(output)})
}
func (h *LabHandler) CommitLabHandler(c *gin.Context) {
	labIDParam := c.Param("id")
	labID, err := strconv.Atoi(labIDParam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
		return
	}

	ctx := c.Request.Context()
	lab, err := h.LabService.GetLab(ctx, uint(labID))
	if err != nil {
		h.Logger.ErrorContext(c, "Error getting lab info", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get lab info"})
		return
	}
	if lab.ContainerName == "" {
		h.Logger.ErrorContext(c, "Lab has no container", "lab_id", labID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lab has no associated container"})
		return
	}

	imageName, err := h.LabService.CommitLab(ctx, lab.ContainerName)
	if err != nil {
		h.Logger.ErrorContext(c, "Error committing container",
			"error", err,
			"container", lab.ContainerName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not commit container"})
		return
	}
	lab.CommitImage = imageName

	h.Logger.InfoContext(c, "Container committed successfully",
		"lab_id", labID,
		"container", lab.ContainerName,
		"image_name", imageName)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Container committed successfully",
		"image_name": imageName,
	})
}
func (h *LabHandler) DeleteCommitLabHandler(c *gin.Context) {
	labIDParam := c.Param("id")
	labID, err := strconv.Atoi(labIDParam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
		return
	}
	ctx := c.Request.Context()
	lab, err := h.LabService.GetLab(ctx, uint(labID))
	if err != nil {
		h.Logger.ErrorContext(c, "Error getting lab info", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get lab info"})
		return
	}
	err = h.LabService.DeleteContainerCommits(ctx, lab.ContainerName)
	if err != nil {
		h.Logger.ErrorContext(c, "Error deleting container", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete container"})
		return
	}
	h.Logger.InfoContext(c, "Container deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Container deleted"})
}
