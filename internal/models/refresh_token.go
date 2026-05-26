package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	Jti       string    `gorm:"uniqueIndex;not null" json:"jti"`
	UserID    string    `gorm:"index;not null" json:"userId"`
	Revoked   bool      `gorm:"default:false" json:"revoked"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (RefreshToken) TableName() string { return "RefreshToken" }

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		r.ID = id.String()
	}
	return nil
}
