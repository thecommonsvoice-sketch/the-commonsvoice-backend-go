package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/validator"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewCategoryHandler(db *gorm.DB, cfg *config.Config) *CategoryHandler {
	return &CategoryHandler{DB: db, Cfg: cfg}
}

type createCategoryRequest struct {
	Name        string  `json:"name" validate:"required,min=2"`
	Description string  `json:"description,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
	ParentID    *string `json:"parentId,omitempty"`
}

type updateCategoryRequest struct {
	Name        string  `json:"name,omitempty" validate:"omitempty,min=2"`
	Description string  `json:"description,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
	ParentID    *string `json:"parentId,omitempty"`
}

var nonAlphaRegex = regexp.MustCompile(`[^a-z0-9]+`)

func generateCategorySlug(name string) string {
	slug := strings.ToLower(name)
	slug = nonAlphaRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"message": "Validation failed", "errors": errs,
		})
		return
	}

	if req.ParentID != nil && *req.ParentID != "" {
		var parent models.Category
		if err := h.DB.First(&parent, "id = ?", *req.ParentID).Error; err != nil {
			response.Error(w, http.StatusBadRequest, "Parent category not found")
			return
		}
	}

	slug := generateCategorySlug(req.Name)
	var count int64
	h.DB.Model(&models.Category{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		slug = fmt.Sprintf("%s-%d", slug, time.Now().UnixMilli())
	}

	slug = strings.TrimRight(slug, "-")
	if len(slug) > 60 {
		slug = slug[:60]
		slug = strings.TrimRight(slug, "-")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	category := models.Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		IsActive:    isActive,
		ParentID:    req.ParentID,
	}

	if err := h.DB.Create(&category).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create category")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"message": "Category created successfully", "category": category,
	})
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	var categories []models.Category
	h.DB.Where("\"isActive\" = ? AND \"parentId\" IS NULL", true).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("\"isActive\" = ?", true).
				Select("id", "name", "slug", "\"isActive\"")
		}).
		Find(&categories)

	// Filter out inactive children
	for i := range categories {
		children := make([]models.Category, 0, len(categories[i].Children))
		for _, c := range categories[i].Children {
			if c.IsActive {
				children = append(children, c)
			}
		}
		categories[i].Children = children
	}

	orderedSlugs := []string{
		"general", "politics", "science-and-technology",
		"sports-and-entertainment", "business",
	}
	orderMap := make(map[string]int)
	for i, s := range orderedSlugs {
		orderMap[s] = i + 1
	}

	sort.Slice(categories, func(i, j int) bool {
		oi := orderMap[categories[i].Slug]
		oj := orderMap[categories[j].Slug]
		if oi == 0 {
			oi = 999
		}
		if oj == 0 {
			oj = 999
		}
		return oi < oj
	})

	response.JSON(w, http.StatusOK, map[string]any{"categories": categories})
}

func (h *CategoryHandler) ListWithHierarchy(w http.ResponseWriter, r *http.Request) {
	var categories []models.Category
	h.DB.Where("\"isActive\" = ?", true).
		Preload("Parent", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "slug")
		}).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("\"isActive\" = ?", true).
				Select("id", "name", "slug", "\"isActive\"")
		}).
		Order("name ASC").
		Find(&categories)

	// Filter out inactive children
	for i := range categories {
		children := make([]models.Category, 0, len(categories[i].Children))
		for _, c := range categories[i].Children {
			if c.IsActive {
				children = append(children, c)
			}
		}
		categories[i].Children = children
	}

	response.JSON(w, http.StatusOK, map[string]any{"categories": categories})
}

func (h *CategoryHandler) ListInactive(w http.ResponseWriter, r *http.Request) {
	var categories []models.Category
	h.DB.Where("\"isActive\" = ?", false).
		Preload("Parent", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name")
		}).
		Order("name ASC").
		Find(&categories)

	response.JSON(w, http.StatusOK, map[string]any{"categories": categories})
}

func (h *CategoryHandler) GetBySlugOrId(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("slugOrId")
	if identifier == "" {
		response.Error(w, http.StatusBadRequest, "Slug or ID is required")
		return
	}

	var category models.Category
	if err := h.DB.Where("id = ? OR slug = ?", identifier, identifier).First(&category).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Category not found")
		return
	}

	if !category.IsActive {
		response.Error(w, http.StatusNotFound, "Category not found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"category": category})
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("slugOrId")
	if identifier == "" {
		response.Error(w, http.StatusBadRequest, "Slug or ID is required")
		return
	}

	var req updateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"message": "Validation failed", "errors": errs,
		})
		return
	}

	var category models.Category
	if err := h.DB.Where("id = ? OR slug = ?", identifier, identifier).First(&category).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Category not found")
		return
	}

	updates := map[string]any{}

	if req.Name != "" {
		updates["name"] = req.Name
		slug := generateCategorySlug(req.Name)
		var count int64
		h.DB.Model(&models.Category{}).Where("slug = ? AND id != ?", slug, category.ID).Count(&count)
		if count > 0 {
			slug = fmt.Sprintf("%s-%d", slug, time.Now().UnixMilli())
		}
		updates["slug"] = slug
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.IsActive != nil {
		updates["isActive"] = *req.IsActive
	}

	if req.ParentID != nil {
		if *req.ParentID != "" {
			var parent models.Category
			if err := h.DB.First(&parent, "id = ?", *req.ParentID).Error; err != nil {
				response.Error(w, http.StatusBadRequest, "Parent category not found")
				return
			}
		}
		updates["parentId"] = *req.ParentID
	}

	if err := h.DB.Model(&category).Updates(updates).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update category")
		return
	}

	h.DB.First(&category, category.ID)
	response.JSON(w, http.StatusOK, map[string]any{
		"message": "Category updated successfully", "category": category,
	})
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("slugOrId")
	if identifier == "" {
		response.Error(w, http.StatusBadRequest, "Slug or ID is required")
		return
	}

	var category models.Category
	if err := h.DB.Where("id = ? OR slug = ?", identifier, identifier).First(&category).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Category not found")
		return
	}

	h.DB.Model(&category).Update("\"isActive\"", false)
	response.JSON(w, http.StatusOK, map[string]string{"message": "Category deleted (soft) successfully"})
}

func (h *CategoryHandler) Restore(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("slugOrId")
	if identifier == "" {
		response.Error(w, http.StatusBadRequest, "Slug or ID is required")
		return
	}

	var category models.Category
	if err := h.DB.Where("id = ? OR slug = ?", identifier, identifier).First(&category).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Category not found")
		return
	}

	h.DB.Model(&category).Update("\"isActive\"", true)
	response.JSON(w, http.StatusOK, map[string]string{"message": "Category restored successfully"})
}

func (h *CategoryHandler) HardDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "ID is required")
		return
	}

	var category models.Category
	if err := h.DB.First(&category, "id = ?", id).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Category not found")
		return
	}

	var articleCount int64
	h.DB.Model(&models.Article{}).Where("\"categoryId\" = ?", id).Count(&articleCount)
	if articleCount > 0 {
		response.Error(w, http.StatusBadRequest,
			"Cannot permanently delete: articles are assigned to this category. Reassign them first.")
		return
	}

	if err := h.DB.Delete(&category).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to permanently delete category")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Category permanently deleted"})
}


