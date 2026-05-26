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
	CoverImage      *string        `gorm:"column:coverImage" json:"coverImage"`
	CategoryID      string         `gorm:"column:categoryId;index;not null" json:"categoryId"`
	AuthorID        string         `gorm:"column:authorId;index;not null" json:"authorId"`
	Status          ArticleStatus  `gorm:"default:DRAFT" json:"status"`
	PublishedAt     *time.Time     `gorm:"column:publishedAt" json:"publishedAt"`
	Excerpt         *string        `json:"excerpt"`
	MetaTitle       *string        `gorm:"column:metaTitle" json:"metaTitle"`
	MetaDescription *string        `gorm:"column:metaDescription" json:"metaDescription"`
	OgImage         *string        `gorm:"column:ogImage" json:"ogImage"`
	Tags            StringArray    `gorm:"type:text[]" json:"tags"`
	DeletedAt       *time.Time     `gorm:"column:deletedAt" json:"deletedAt"`
	CreatedAt       time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updatedAt" json:"updatedAt"`

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
