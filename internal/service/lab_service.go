package service

import (
	"context"
	"fmt"
	"lab/internal/interfaces"
	"lab/internal/model"
	"log/slog"
	"os/exec"
)

type LabService struct {
	LabRepository interfaces.LabInterface
	Logger        *slog.Logger
}

func NewLabService(labRepository interfaces.LabInterface, logger *slog.Logger) *LabService {
	return &LabService{
		LabRepository: labRepository,
		Logger:        logger,
	}
}
func (s *LabService) CreateLab(ctx context.Context, lab *model.Lab) error {
	if err := s.LabRepository.CreateLab(ctx, lab); err != nil {
		s.Logger.ErrorContext(ctx, "Error while creating lab", "error", err)
		return err
	}
	s.Logger.InfoContext(ctx, "Lab created successfully")
	return nil
}
func (s *LabService) UpdateLab(ctx context.Context, lab *model.Lab) error {
	if err := s.LabRepository.UpdateLab(ctx, lab); err != nil {
		s.Logger.ErrorContext(ctx, "Error while updating lab", "error", err)
		return err
	}
	s.Logger.InfoContext(ctx, "Lab updated successfully")
	return nil
}
func (s *LabService) DeleteLab(ctx context.Context, id int) error {
	if err := s.LabRepository.DeleteLab(ctx, id); err != nil {
		s.Logger.ErrorContext(ctx, "Error while deleting lab", "error", err)
		return err
	}
	s.Logger.InfoContext(ctx, "Lab deleted successfully")
	return nil
}
func (s *LabService) GetLab(ctx context.Context, id int) (*model.Lab, error) {
	lab, err := s.LabRepository.GetLab(ctx, id)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting lab", "error", err)
		return nil, err
	}
	s.Logger.InfoContext(ctx, "Lab found successfully")
	return lab, nil
}
func (s *LabService) StartLab(ctx context.Context, id int) (string, error) {
	lab, err := s.LabRepository.GetLab(ctx, id)
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while getting lab", "error", err)
		return "", err
	}

	if lab.VMImage.Id == 0 {
		return "", fmt.Errorf("лаборатория %d не имеет связанного образа ВМ", id)
	}

	containerName := fmt.Sprintf("lab_%d", id)
	cmd := exec.Command("docker", "run", "-dit", "--name", containerName, lab.VMImage.FilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while starting lab", "error", err)
		return "", fmt.Errorf("ошибка запуска контейнера: %s", err)
	}

	s.Logger.InfoContext(ctx, "Lab started successfully", "container_id", string(output))
	return string(output), nil
}
func (s *LabService) StopLab(ctx context.Context, id int) error {
	containerName := fmt.Sprintf("lab_%d", id)
	cmd := exec.Command("docker", "stop", containerName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while stopping lab", "error", err)
		return fmt.Errorf("ошибка остановки контейнера: %s", err)
	}

	cmd = exec.Command("docker", "rm", containerName)
	_, err = cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while removing lab container", "error", err)
		return fmt.Errorf("ошибка удаления контейнера: %s", err)
	}

	s.Logger.InfoContext(ctx, "Lab stopped and removed successfully")
	return nil
}
func (s *LabService) ExecuteCommand(ctx context.Context, id int, command []string) (string, error) {
	containerName := fmt.Sprintf("lab_%d", id)
	cmd := exec.Command("docker", append([]string{"exec", containerName}, command...)...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Logger.ErrorContext(ctx, "Error while executing command", "error", err)
		return "", fmt.Errorf("ошибка выполнения команды: %s", err)
	}

	return string(output), nil
}
