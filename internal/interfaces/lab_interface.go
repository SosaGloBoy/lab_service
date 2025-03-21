package interfaces

import (
	"context"
	"lab/internal/model"
)

type LabInterface interface {
	CreateLab(ctx context.Context, lab *model.Lab) error
	UpdateLab(ctx context.Context, lab *model.Lab) error
	DeleteLab(ctx context.Context, id int) error
	GetLab(ctx context.Context, id int) (*model.Lab, error)
	GetAllLabs(ctx context.Context) ([]*model.Lab, error)
}
