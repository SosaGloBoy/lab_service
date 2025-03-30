package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"lab/internal/interfaces"
	"lab/internal/model"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
)

type LabService struct {
	LabRepository  interfaces.LabInterface
	TaskServiceURL string // URL для доступа к сервису заданий
	Logger         *slog.Logger
}

func NewLabService(labRepository interfaces.LabInterface, taskServiceURL string, logger *slog.Logger) *LabService {
	return &LabService{
		LabRepository:  labRepository,
		TaskServiceURL: taskServiceURL,
		Logger:         logger,
	}
}

func (s *LabService) CreateLab(ctx context.Context, lab *model.Lab) error {
	// Получаем Docker-образ для задания из API сервиса заданий
	vmImagePath, err := s.getTaskDockerImage(lab.TaskID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Failed to get task Docker image", "error", err)
		return err
	}
	if lab.ID == 0 {
		lab.ID = 1
	}
	containerName := fmt.Sprintf("lab_%d", lab.ID)

	cmd := exec.Command("docker", "run", "-dit", "--name", containerName, vmImagePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while creating container", "error", err)
		return fmt.Errorf("error while creating container %s: %s", containerName, err)
	}
	containerID := string(output) // Преобразуем вывод в строку

	lab.ContainerID = containerID
	return s.LabRepository.CreateLab(ctx, lab)
}

// Метод для получения Docker-образа для задания
func (s *LabService) getTaskDockerImage(taskID uint) (string, error) {
	url := fmt.Sprintf("%s/tasks/%d", s.TaskServiceURL, taskID)

	resp, err := http.Get(url)
	if err != nil {
		s.Logger.Error("Error checking task existence", "error", err)
		return "", fmt.Errorf("error checking task existence: %w", err)
	}
	defer resp.Body.Close()
	s.Logger.Info("Received response from task service", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("task not found")
	}

	var response struct {
		Task struct {
			VMImagePath string `json:"vm_image_path"`
		} `json:"task"` // Мы получаем вложенный объект task
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Error("Error reading task response body", "error", err)
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	s.Logger.Info("Response body", "body", string(body))

	if err := json.Unmarshal(body, &response); err != nil {
		s.Logger.Error("Error unmarshaling task response", "error", err)
		return "", fmt.Errorf("error unmarshaling task response: %w", err)
	}

	return response.Task.VMImagePath, nil
}

// Метод для запуска контейнера
func (s *LabService) StartLab(ctx context.Context, lab *model.Lab, vmImagePath string) (string, error) {
	if lab.ID == 0 {
		lab.ID = 1
	}
	containerName := fmt.Sprintf("lab_%d", lab.ID)
	cmd := exec.Command("docker", "start", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while starting container", "error", err)
		return "", fmt.Errorf("error while starting container %s: %s", containerName, err)
	}

	s.Logger.InfoContext(ctx, "Container started successfully", "container_id", string(output))
	return string(output), nil
}

// Метод для остановки лаборатории (удаления контейнера)
func (s *LabService) StopLab(ctx context.Context, labID int) error {
	lab, err := s.LabRepository.GetLab(ctx, labID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting lab", "error", err)
		return err
	}

	cmd := exec.Command("docker", "stop", strings.TrimSpace(lab.ContainerID))
	_, err = cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while stopping container", "error", err)
		return fmt.Errorf("error while stopping container %s: %s", lab.ContainerID, err)
	}

	s.Logger.InfoContext(ctx, "Lab stopped successfully", "lab_id", lab.ID)
	return nil
}

// Метод для обновления лаборатории
func (s *LabService) UpdateLab(ctx context.Context, lab *model.Lab) error {
	// Здесь мы просто обновляем лабораторию в базе данных
	if err := s.LabRepository.UpdateLab(ctx, lab); err != nil {
		s.Logger.ErrorContext(ctx, "Error while updating lab", "error", err, "lab_id", lab.ID)
		return err
	}
	s.Logger.InfoContext(ctx, "Lab updated successfully", "lab", lab)
	return nil
}

// Метод для выполнения команд внутри контейнера
func (s *LabService) ExecuteCommand(ctx context.Context, containerID string, command []string) (string, error) {
	cmd := exec.Command("docker", append([]string{"exec", containerID}, command...)...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while executing command inside container", "error", err, "container_id", containerID)
		return "", fmt.Errorf("error while executing command inside container %s: %s", containerID, err)
	}

	s.Logger.InfoContext(ctx, "Command executed successfully in container", "container_id", containerID, "output", string(output))
	return string(output), nil
}

// Метод для получения всех лабораторий
func (s *LabService) GetAllLabs(ctx context.Context) ([]*model.Lab, error) {
	labs, err := s.LabRepository.GetAllLabs(ctx)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting all labs", "error", err)
		return nil, err
	}
	s.Logger.InfoContext(ctx, "All labs retrieved successfully", "labs_count", len(labs))
	return labs, nil
}

// Метод для удаления лаборатории (удаление лаборатории и контейнера)
func (s *LabService) DeleteLab(ctx context.Context, labID int) error {
	// Получаем лабораторию из базы данных
	lab, err := s.LabRepository.GetLab(ctx, labID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting lab", "error", err)
		return err
	}

	// Останавливаем и удаляем контейнер, связанный с лабораторией
	cmdStop := exec.Command("docker", "stop", strings.TrimSpace(lab.ContainerID))
	_, err = cmdStop.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while stopping container", "error", err, "container_id", lab.ContainerID)
		return fmt.Errorf("error while stopping container %s: %s", lab.ContainerID, err)
	}

	cmdRemove := exec.Command("docker", "rm", strings.TrimSpace(lab.ContainerID))
	_, err = cmdRemove.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while removing container", "error", err, "container_id", lab.ContainerID)
		return fmt.Errorf("error while removing container %s: %s", lab.ContainerID, err)
	}

	// Удаляем лабораторию из базы данных
	if err := s.LabRepository.DeleteLab(ctx, labID); err != nil {
		s.Logger.ErrorContext(ctx, "Error while deleting lab", "error", err, "lab_id", labID)
		return err
	}

	s.Logger.InfoContext(ctx, "Lab deleted successfully", "lab_id", labID)
	return nil
}
func (s *LabService) GetLab(ctx context.Context, labID uint) (*model.Lab, error) {

	lab, err := s.LabRepository.GetLab(ctx, int(labID))
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting lab", "error", err, "lab_id", labID)
		return nil, fmt.Errorf("failed to get lab: %w", err)
	}

	s.Logger.InfoContext(ctx, "Lab found successfully", "lab_id", lab.ID)
	return lab, nil
}
