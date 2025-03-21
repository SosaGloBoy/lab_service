package model

import "time"

type Lab struct {
	ID          uint      `gorm:"primary_key" json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	VmImageId   int       `json:"vm_image_id"`
	VMImage     VMImage   `gorm:"foreignKey:VmImageID" json:"vm_image"`
	CreatedAt   time.Time `json:"created_at"`
}
