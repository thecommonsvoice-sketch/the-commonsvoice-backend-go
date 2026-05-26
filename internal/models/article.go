package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Article struct {
	ID              string         `gorm:"primaryKey;type:text" json:"id"`
	Title           string         `gorm:"not null" json:"title"`
	Slug            string         `gorm:"uniqueIndex;not null" json:"slug"`
	Content         string         `gorm:"not null" json:"content"`
	CoverImage      *string        `json:"coverImage"`
	CategoryID      string         `gorm:"index;not null" json:"categoryId"`
	AuthorID        string         `gorm:"index;not null" json:"authorId"`
	Status          ArticleStatus  `gorm:"default:DRAFT" json:"status"`
	PublishedAt     *time.Time     `json:"publishedAt"`
	Excerpt         *string        `json:"excerpt"`
	MetaTitle       *string        `json:"metaTitle"`
	MetaDescription *string        `json:"metaDescription"`
	OgImage         *string        `json:"ogImage"`
	Tags            StringArray    `gorm:"type:text[]" json:"tags"`
	DeletedAt       *time.Time     `json:"deletedAt"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`

	Category Category      `gorm:"foreignKey:CategoryID" json:"category"`
	Author   User          `gorm:"foreignKey:AuthorID" json:"author"`
	Videos   []ArticleVideo `gorm:"foreignKey:ArticleID" json:"videos,omitempty"`
}

func (Article) TableName() string { return "Article" }

func (a *Article) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		a.ID = id.String()
	}
	return nil
}
