package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ArticleVideo struct {
	ID          string    `gorm:"primaryKey;type:text" json:"id"`
	ArticleID   string    `gorm:"column:articleId;index;not null" json:"articleId"`
	Type        string    `gorm:"not null" json:"type"`
	URL         string    `gorm:"not null" json:"url"`
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	Article Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

func (ArticleVideo) TableName() string { return "ArticleVideo" }

func (a *ArticleVideo) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		a.ID = id.String()
	}
	return nil
}
