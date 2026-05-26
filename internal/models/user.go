package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Name      string    `json:"name"`
	Role      Role      `gorm:"default:USER" json:"role"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (User) TableName() string { return "User" }

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		u.ID = id.String()
	}
	return nil
}
