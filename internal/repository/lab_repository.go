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
func (r *LabRepository) CreateLab(ctx context.Context, lab *model.Lab) error {
	if err := r.DB.Create(lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error while creating lab", err)
		return err
	}
	r.Logger.DebugContext(ctx, "Lab created")
	return nil
}
func (r *LabRepository) GetLab(ctx context.Context, id int) (*model.Lab, error) {
	var lab model.Lab
	if err := r.DB.Preload("VmImage").Where("id = ?", id).First(&lab).Error; err != nil {
		r.Logger.WarnContext(ctx, "Can not find lab by id", "id", id, "error", err)
		return nil, err
	}
	r.Logger.DebugContext(ctx, "Lab was found successfully", "id", id)
	return &lab, nil
}
func (r *LabRepository) UpdateLab(ctx context.Context, lab *model.Lab) error {
	if err := r.DB.Save(lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error while updating lab", "id", lab.ID, err)
		return err
	}
	return nil
}
func (r *LabRepository) DeleteLab(ctx context.Context, id int) error {
	var lab model.Lab
	if err := r.DB.Where("id = ?", id).First(&lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error finding laboratory to delete", "id", id, "error", err)
		return err
	}
	if err := r.DB.Delete(&lab).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error deleting laboratory to delete", "id", id, "error", err)
		return err
	}
	r.Logger.InfoContext(ctx, "Lab deleted successfully", "id", id)
	return nil
}
func (r *LabRepository) GetAllLabs(ctx context.Context) ([]*model.Lab, error) {
	var labs []*model.Lab
	if err := r.DB.Preload("VmImage").Find(&labs).Error; err != nil {
		r.Logger.ErrorContext(ctx, "Error finding labs", "error", err)
		return nil, err
	}
	r.Logger.DebugContext(ctx, "Labs found")
	return labs, nil
}
