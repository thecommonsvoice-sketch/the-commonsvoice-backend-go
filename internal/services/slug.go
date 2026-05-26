package services

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"gorm.io/gorm"
)

var nonAlphaRegex = regexp.MustCompile(`[^a-z0-9]+`)

func CreateSlug(title string, excludeId string, db *gorm.DB) (string, error) {
	slug := strings.ToLower(title)
	slug = nonAlphaRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	if len(slug) > 60 {
		slug = slug[:60]
	}
	slug = strings.TrimRight(slug, "-")

	var count int64
	query := db.Model(&models.Article{}).Where("slug = ?", slug)
	if excludeId != "" {
		query = query.Where("id != ?", excludeId)
	}

	if err := query.Count(&count).Error; err != nil {
		return "", err
	}

	if count > 0 {
		slug = fmt.Sprintf("%s-%d", slug, time.Now().UnixMilli())
	}
	return slug, nil
}

