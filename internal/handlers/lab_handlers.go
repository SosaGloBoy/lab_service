package handlers

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"lab/internal/model"
	"lab/internal/service"
	"log/slog"
	"net/http"
	"strconv"
)

type LabHandler struct {
	LabService *service.LabService
	Logger     *slog.Logger
}

func NewLabHandler(labService *service.LabService, logger *slog.Logger) *LabHandler {
	return &LabHandler{
		LabService: labService,
		Logger:     logger,
	}
}
func (h *LabHandler) CreateLabHandler(c *gin.Context) {
	var lab model.Lab
	if err := c.ShouldBindJSON(&lab); err != nil {
		h.Logger.ErrorContext(c, "Failed to bind lab data", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Lab Data"})
		return
	}
	ctx := c.Request.Context()
	err := h.LabService.CreateLab(ctx, &lab)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to create lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lab"})
		return
	}
	h.Logger.InfoContext(c, "Lab created", "lab", lab)
	c.JSON(http.StatusOK, gin.H{"message": "Laboratory created successfully", "lab_id": lab.ID})
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
	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}
func (h *LabHandler) DeleteLabHandler(c *gin.Context) {
	labIDparam := c.Param("id")
	labID, err := strconv.Atoi(labIDparam)
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
	h.Logger.InfoContext(c, "Lab deleted", "lab", labID)
	c.JSON(http.StatusOK, gin.H{"message": "Laboratory deleted successfully"})
}
func (h *LabHandler) GetLabsHandler(c *gin.Context) {
	labIDparam := c.Param("id")
	labID, err := strconv.Atoi(labIDparam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
		return
	}
	ctx := c.Request.Context()
	lab, err := h.LabService.GetLab(ctx, labID)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to get lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get lab"})
		return
	}
	h.Logger.InfoContext(c, "Lab found", "lab", lab)
	c.JSON(http.StatusOK, gin.H{"lab": lab})
}
func (h *LabHandler) StartLabHandler(c *gin.Context) {
	labIDparam := c.Param("id")
	labID, err := strconv.Atoi(labIDparam)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to parse lab id", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab id"})
	}
	containerID, err := h.LabService.StartLab(context.Background(), labID)
	if err != nil {
		h.Logger.ErrorContext(c, "Failed to start lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start lab"})
		return
	}
	h.Logger.InfoContext(c, "Lab started successfully", "containerID", containerID)
	c.JSON(http.StatusOK, gin.H{"message": "Laboratory started successfully"})
}
func (h *LabHandler) StopLabHandler(c *gin.Context) {
	labID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab ID"})
		return
	}

	err = h.LabService.StopLab(context.Background(), labID)
	if err != nil {
		h.Logger.ErrorContext(context.Background(), "Error stopping lab", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not stop lab"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Lab stopped"})
}
func (h *LabHandler) ExecuteCommand(c *gin.Context) {
	labID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lab ID"})
		return
	}

	var request struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	output, err := h.LabService.ExecuteCommand(context.Background(), labID, []string{"/bin/sh", "-c", request.Command})
	if err != nil {
		h.Logger.ErrorContext(context.Background(), "Error executing command", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not execute command"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"output": output})
}
