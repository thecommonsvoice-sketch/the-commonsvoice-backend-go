package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bookmark struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	UserID    string    `gorm:"index;uniqueIndex:idx_user_article;not null" json:"userId"`
	ArticleID string    `gorm:"index;uniqueIndex:idx_user_article;not null" json:"articleId"`
	CreatedAt time.Time `json:"createdAt"`

	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

func (Bookmark) TableName() string { return "Bookmark" }

func (b *Bookmark) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		b.ID = id.String()
	}
	return nil
}
