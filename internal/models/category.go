package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID          string    `gorm:"primaryKey;type:text" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Description string    `json:"description"`
	IsActive    bool      `gorm:"column:isActive;default:true" json:"isActive"`
	ParentID    *string   `gorm:"column:parentId;index" json:"parentId"`
	CreatedAt   time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (Category) TableName() string { return "Category" }

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		c.ID = id.String()
	}
	return nil
}
