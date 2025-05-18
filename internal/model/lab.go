package model

import "time"

type Lab struct {
	ID            uint      `gorm:"primary_key" json:"id"`
	Title         string    `json:"title"`        // Название лаборатории
	TaskID        uint      `json:"task_id"`      // Ссылка на задание (task_id из сервиса заданий)
	CreatedAt     time.Time `json:"created_at"`   // Время создания лаборатории
	UpdatedAt     time.Time `json:"updated_at"`   // Время последнего обновления лаборатории
	ContainerID   string    `json:"container_id"` // ID Docker контейнера
	ContainerName string    `json:"container_name"`
	AccessURL     string    `json:"access_url"`
	CommitImage   string    `json:"commit_image"`
}
