package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Comment struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	Content   string    `gorm:"not null" json:"content"`
	UserID    string    `gorm:"column:userId;index;not null" json:"userId"`
	ArticleID string    `gorm:"column:articleId;index;not null" json:"articleId"`
	ParentID  *string   `gorm:"column:parentId;index" json:"parentId"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt" json:"updatedAt"`

	User    User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article Article   `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
	Parent  *Comment  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Replies []Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

func (Comment) TableName() string { return "Comment" }

func (c *Comment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		c.ID = id.String()
	}
	return nil
}
