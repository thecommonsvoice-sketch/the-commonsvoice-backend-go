package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LatestNews struct {
	ID          string    `gorm:"primaryKey;type:text" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	PhotoURL    *string   `gorm:"column:photoUrl" json:"photoUrl"`
	Link        *string   `json:"link"`
	Type        *string   `json:"type"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updatedAt" json:"updatedAt"`
}

func (LatestNews) TableName() string { return "LatestNews" }

func (l *LatestNews) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		l.ID = id.String()
	}
	return nil
}
