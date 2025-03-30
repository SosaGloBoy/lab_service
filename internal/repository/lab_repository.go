package repository

import (
	"context"
	"gorm.io/gorm"
	"lab/internal/interfaces"
	"lab/internal/model"
	"log/slog"
)

type LabRepository struct {
	DB     *gorm.DB
	Logger *slog.Logger
}

func NewLabRepository(db *gorm.DB, logger *slog.Logger) interfaces.LabInterface {
	return &LabRepository{
		DB:     db,
		Logger: logger,
	}
}

// Метод для создания лаборатории (запуска контейнера)
func (r *LabRepository) CreateLab(ctx context.Context, lab *model.Lab) error {
	if err := r.DB.Create(lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error while creating lab", "error", err)
		return err
	}
	r.Logger.InfoContext(ctx, "Lab created successfully", "lab", lab)
	return nil
}

// Метод для обновления лаборатории
func (r *LabRepository) UpdateLab(ctx context.Context, lab *model.Lab) error {
	if err := r.DB.Save(lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error while updating lab", "error", err, "lab_id", lab.ID)
		return err
	}
	r.Logger.InfoContext(ctx, "Lab updated successfully", "lab", lab)
	return nil
}

// Метод для удаления лаборатории
func (r *LabRepository) DeleteLab(ctx context.Context, id int) error {
	var lab model.Lab
	if err := r.DB.Where("id = ?", id).First(&lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error finding lab to delete", "error", err, "lab_id", id)
		return err
	}
	if err := r.DB.Delete(&lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error deleting lab", "error", err, "lab_id", id)
		return err
	}
	r.Logger.InfoContext(ctx, "Lab deleted successfully", "lab_id", id)
	return nil
}

// Метод для получения лаборатории по ID
func (r *LabRepository) GetLab(ctx context.Context, id int) (*model.Lab, error) {
	var lab model.Lab
	// Получаем лабораторию по ID
	if err := r.DB.Where("id = ?", id).First(&lab).Error; err != nil {
		r.Logger.WarnContext(ctx, "Can not find lab by id", "lab_id", id, "error", err)
		return nil, err
	}
	r.Logger.InfoContext(ctx, "Lab found successfully", "lab", lab)
	return &lab, nil
}

// Метод для получения всех лабораторий
func (r *LabRepository) GetAllLabs(ctx context.Context) ([]*model.Lab, error) {
	var labs []*model.Lab
	// Получаем все лаборатории
	if err := r.DB.Find(&labs).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error finding labs", "error", err)
		return nil, err
	}
	r.Logger.InfoContext(ctx, "Labs found", "labs_count", len(labs))
	return labs, nil
}
