package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"lab/internal/interfaces"
	"lab/internal/model"
	"lab/utils"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"
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

func (s *LabService) CreateLab(ctx context.Context, taskID uint, vmImagePath string) (string, string, uint, error) {
	containerName := fmt.Sprintf("lab_%d_%s", taskID, time.Now().Format("20060102_150405_999"))
	freePort, err := utils.GetFreePort()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error getting free port", "error", err)
		return "", "", 0, fmt.Errorf("failed to get free port: %w", err)
	}
	cmd := exec.CommandContext(
		ctx,
		"docker", "run", "-dit",
		"--name", containerName,
		"-p", fmt.Sprintf("%d:7681", freePort),
		vmImagePath,
	)

	s.Logger.DebugContext(ctx, "Running command", "cmd", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while creating container", "error", err, "output", string(output))
		return "", "", 0, fmt.Errorf("error while creating container %s: %w (output: %s)", containerName, err, string(output))
	}
	containerID := strings.TrimSpace(string(output))

	accessURL := fmt.Sprintf("http://localhost:%d", freePort)

	lab := &model.Lab{
		TaskID:        taskID,
		ContainerID:   containerID,
		ContainerName: containerName,
		AccessURL:     accessURL,
		CommitImage:   vmImagePath,
	}
	if err := s.LabRepository.CreateLab(ctx, lab); err != nil {
		s.Logger.ErrorContext(ctx, "Failed to save lab to database", "error", err)
		return "", "", 0, fmt.Errorf("failed to save lab to database: %w", err)
	}
	s.Logger.DebugContext(ctx, "Lab created", "id", lab.ID)

	return containerID, accessURL, lab.ID, nil
}

func (s *LabService) CreateLabFromCommit(ctx context.Context, lab *model.Lab, imageName string) error {
	lab.ContainerName = fmt.Sprintf("lab_%d_%s", lab.TaskID, time.Now().Format("20060102_150405_999"))
	freePort, err := utils.GetFreePort()
	if err != nil {
		return fmt.Errorf("failed to get free port: %w", err)
	}
	cmd := exec.CommandContext(
		ctx,
		"docker", "run", "-dit",
		"--name", lab.ContainerName,
		"-p", fmt.Sprintf("%d:3000", freePort),
		imageName,
		"--base", "/wetty",
		"--reverse-proxy",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run container: %w (output: %s)", err, string(output))
	}

	lab.ContainerID = strings.TrimSpace(string(output))
	lab.AccessURL = fmt.Sprintf("http://localhost:%d", freePort)

	return s.LabRepository.CreateLab(ctx, lab)
}

func (s *LabService) getTaskDockerImage(ctx context.Context, taskID uint) (string, error) {
	url := fmt.Sprintf("%s/tasks/%d", s.TaskServiceURL, taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error creating request for task service", "error", err)
		return "", fmt.Errorf("error creating request: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error executing request to task service", "error", err)
		return "", fmt.Errorf("error checking task existence: %w", err)
	}
	defer resp.Body.Close()

	s.Logger.InfoContext(ctx, "Received response from task service", "status_code", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("task not found")
	}

	var response struct {
		Task struct {
			VMImagePath string `json:"vm_image_path"`
		} `json:"task"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error reading task response body", "error", err)
		return "", fmt.Errorf("error reading response body: %w", err)
	}
	s.Logger.DebugContext(ctx, "Task service response body", "body", string(body))

	if err := json.Unmarshal(body, &response); err != nil {
		s.Logger.ErrorContext(ctx, "Error unmarshaling task response", "error", err)
		return "", fmt.Errorf("error unmarshaling task response: %w", err)
	}

	return response.Task.VMImagePath, nil
}

func (s *LabService) StartLab(ctx context.Context, lab *model.Lab, vmImagePath string) (string, error) {
	currLab, err := s.GetLab(ctx, lab.ID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error getting lab", "error", err)
		return "", fmt.Errorf("failed to get lab: %w", err)
	}
	containerName := currLab.ContainerName
	cmd := exec.CommandContext(ctx, "docker", "start", containerName)
	s.Logger.DebugContext(ctx, "Running command", "cmd", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while starting container", "error", err, "output", string(output))
		return "", fmt.Errorf("error while starting container %s: %w (output: %s)", containerName, err, string(output))
	}

	s.Logger.InfoContext(ctx, "Container started successfully", "container_id", strings.TrimSpace(string(output)))
	return strings.TrimSpace(string(output)), nil
}

func (s *LabService) StopLab(ctx context.Context, labID int) error {
	lab, err := s.LabRepository.GetLab(ctx, labID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting lab", "error", err)
		return fmt.Errorf("failed to get lab: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker", "stop", strings.TrimSpace(lab.ContainerID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while stopping container", "error", err, "output", string(output))
		return fmt.Errorf("error while stopping container %s: %w (output: %s)", lab.ContainerID, err, string(output))
	}

	s.Logger.InfoContext(ctx, "Lab stopped successfully", "lab_id", lab.ID)
	return nil
}

func (s *LabService) UpdateLab(ctx context.Context, lab *model.Lab) error {
	if err := s.LabRepository.UpdateLab(ctx, lab); err != nil {
		s.Logger.ErrorContext(ctx, "Error while updating lab", "error", err, "lab_id", lab.ID)
		return fmt.Errorf("failed to update lab %d: %w", lab.ID, err)
	}
	s.Logger.InfoContext(ctx, "Lab updated successfully", "lab", lab)
	return nil
}

func (s *LabService) ExecuteCommand(ctx context.Context, containerID string, command []string) (string, error) {
	// Объединяем команду в одну строку для исполнения через shell
	shellCommand := strings.Join(command, " ")

	// Используем sh -c "команда" для запуска через shell
	args := []string{"exec", containerID, "sh", "-c", shellCommand}
	cmd := exec.CommandContext(ctx, "docker", args...)

	s.Logger.DebugContext(ctx, "Executing command in container", "cmd", cmd.String())

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		s.Logger.ErrorContext(ctx, "Error executing command inside container",
			"error", err, "container_id", containerID, "output", outputStr)
		return outputStr, fmt.Errorf("error executing command in container %s: %w (output: %s)", containerID, err, outputStr)
	}

	s.Logger.InfoContext(ctx, "Command executed successfully in container", "container_id", containerID, "output", outputStr)
	return outputStr, nil
}

func (s *LabService) GetAllLabs(ctx context.Context) ([]*model.Lab, error) {
	labs, err := s.LabRepository.GetAllLabs(ctx)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting all labs", "error", err)
		return nil, fmt.Errorf("failed to get all labs: %w", err)
	}
	s.Logger.InfoContext(ctx, "All labs retrieved successfully", "labs_count", len(labs))
	return labs, nil
}

func (s *LabService) DeleteLab(ctx context.Context, labID int) error {
	lab, err := s.LabRepository.GetLab(ctx, labID)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting lab", "error", err)
		return fmt.Errorf("failed to get lab: %w", err)
	}

	cmdStop := exec.CommandContext(ctx, "docker", "stop", strings.TrimSpace(lab.ContainerID))
	outputStop, err := cmdStop.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while stopping container", "error", err, "container_id", lab.ContainerID, "output", string(outputStop))
		return fmt.Errorf("error while stopping container %s: %w (output: %s)", lab.ContainerID, err, string(outputStop))
	}

	cmdRemove := exec.CommandContext(ctx, "docker", "rm", strings.TrimSpace(lab.ContainerID))
	outputRemove, err := cmdRemove.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while removing container", "error", err, "container_id", lab.ContainerID, "output", string(outputRemove))
		return fmt.Errorf("error while removing container %s: %w (output: %s)", lab.ContainerID, err, string(outputRemove))
	}
	if err := s.LabRepository.DeleteLab(ctx, labID); err != nil {
		s.Logger.ErrorContext(ctx, "Error while deleting lab", "error", err, "lab_id", labID)
		return fmt.Errorf("failed to delete lab %d: %w", labID, err)
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

func (s *LabService) CommitLab(ctx context.Context, containerName string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	newImageName := fmt.Sprintf("%s-snapshot-%s", containerName, timestamp)

	checkCmd := exec.CommandContext(ctx, "docker", "inspect", "--format={{.State.Running}}", containerName)
	output, err := checkCmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Container check failed",
			"container", containerName,
			"error", err,
			"output", string(output))
		return "", fmt.Errorf("container %s not found: %w (output: %s)", containerName, err, string(output))
	}

	if strings.TrimSpace(string(output)) != "true" {
		err := fmt.Errorf("container %s is not running", containerName)
		s.Logger.ErrorContext(ctx, "Container not running", "error", err)
		return "", err
	}
	if !strings.Contains(newImageName, ":") {
		newImageName += ":latest"
	}

	commitCmd := exec.CommandContext(ctx, "docker", "commit",
		"-a", "lab-system",
		"-m", fmt.Sprintf("Autocommit of %s at %s", containerName, timestamp),
		containerName,
		newImageName)
	s.Logger.DebugContext(ctx, "Executing commit command",
		"cmd", commitCmd.String(),
		"new_image", newImageName)

	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Commit failed",
			"error", err,
			"output", string(commitOutput))
		return "", fmt.Errorf("commit failed: %w (output: %s)", err, string(commitOutput))
	}

	s.Logger.InfoContext(ctx, "Container committed successfully",
		"container", containerName,
		"new_image", newImageName)

	return newImageName, nil
}
func (s *LabService) DeleteContainerCommits(ctx context.Context, containerName string) error {
	imagePattern := containerName + "-snapshot-*"

	cmdListImages := exec.CommandContext(ctx, "docker", "images", "-q", imagePattern)
	output, err := cmdListImages.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error listing images: %w", err)
	}
	imageIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(imageIDs) == 0 || (len(imageIDs) == 1 && imageIDs[0] == "") {
		s.Logger.DebugContext(ctx, "No commits found for container",
			"container_name", containerName)
		return nil
	}
	var lastError error
	for _, imageID := range imageIDs {
		if imageID == "" {
			continue
		}

		cmdRemoveImage := exec.CommandContext(ctx, "docker", "rmi", "-f", imageID)
		if _, err := cmdRemoveImage.CombinedOutput(); err != nil {
			s.Logger.WarnContext(ctx, "Failed to delete image",
				"image_id", imageID,
				"error", err)
			lastError = err
		}
	}

	if lastError != nil {
		return fmt.Errorf("some images were not deleted, last error: %w", lastError)
	}

	s.Logger.InfoContext(ctx, "Successfully deleted all container commits",
		"container_name", containerName,
		"deleted_count", len(imageIDs))
	return nil
}
