package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/services"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/validator"
	"gorm.io/gorm"
)

type AdminHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewAdminHandler(db *gorm.DB, cfg *config.Config) *AdminHandler {
	return &AdminHandler{DB: db, Cfg: cfg}
}

func (h *AdminHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}
	search := r.URL.Query().Get("search")
	offset := (page - 1) * limit

	db := h.DB.Model(&models.User{}).Select("id", "email", "name", "role", "\"isActive\"", "\"createdAt\"", "\"updatedAt\"")
	if search != "" {
		like := "%" + search + "%"
		db = db.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}

	var total int64
	db.Count(&total)

	var users []models.User
	db.Offset(offset).Limit(limit).Order("\"createdAt\" DESC").Find(&users)
	if len(users) == 0 {
		users = []models.User{}
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"users": users,
		"pagination": map[string]any{
			"total": total, "page": page, "limit": limit, "totalPages": totalPages,
		},
	})
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name" validate:"required,min=2"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=6"`
		Role     string `json:"role,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid user data", "errors": errs})
		return
	}

	var existing int64
	h.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&existing)
	if existing > 0 {
		response.Error(w, http.StatusConflict, "User with this email already exists")
		return
	}

	hashedPassword, err := services.HashPassword(req.Password)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	role := models.RoleUser
	if req.Role != "" {
		role = models.Role(req.Role)
	}

	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     role,
		IsActive: true,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"message": "User created successfully", "user": user})
}

func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	if userId == "" {
		response.Error(w, http.StatusBadRequest, "User ID is required")
		return
	}

	var req struct {
		Role string `json:"role" validate:"required,oneof=ADMIN EDITOR REPORTER USER"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid role", "errors": errs})
		return
	}

	result := h.DB.Model(&models.User{}).Where("id = ?", userId).Update("role", req.Role)
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update user role")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "User not found")
		return
	}

	var user models.User
	h.DB.First(&user, "id = ?", userId)
	response.JSON(w, http.StatusOK, map[string]any{"message": "User role updated successfully", "user": user})
}

func (h *AdminHandler) ToggleUserActiveStatus(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	if userId == "" {
		response.Error(w, http.StatusBadRequest, "User ID is required")
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", userId).Error; err != nil {
		response.Error(w, http.StatusNotFound, "User not found")
		return
	}

	user.IsActive = !user.IsActive
	if err := h.DB.Save(&user).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to toggle user status")
		return
	}

	status := "activated"
	if !user.IsActive {
		status = "deactivated"
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"message": "User " + status, "user": user,
	})
}

func (h *AdminHandler) GetArticleBySlugOrId(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("slugOrId")
	if identifier == "" {
		response.Error(w, http.StatusBadRequest, "Slug or ID is required")
		return
	}

	var article models.Article
	if err := h.DB.Unscoped().Preload("Author", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "email")
	}).Preload("Category", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "slug")
	}).Where("id = ? OR slug = ?", identifier, identifier).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"article": article})
}

func (h *AdminHandler) GetAllArticles(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}
	search := r.URL.Query().Get("search")
	offset := (page - 1) * limit

	db := h.DB.Model(&models.Article{}).Unscoped()
	if search != "" {
		db = db.Where("title ILIKE ?", "%"+search+"%")
	}

	var total int64
	db.Count(&total)

	var articles []models.Article
	db.Select("id", "title", "slug", "\"coverImage\"", "status", "\"authorId\"", "\"categoryId\"", "\"createdAt\"", "\"updatedAt\"", "\"publishedAt\"", "\"deletedAt\"").
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "email")
		}).
		Preload("Category", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "slug")
		}).
		Offset(offset).Limit(limit).Order("\"createdAt\" DESC").
		Find(&articles)
	if len(articles) == 0 {
		articles = []models.Article{}
	}

	today := time.Now().Truncate(24 * time.Hour)
	var publishedTodayCount int64
	h.DB.Model(&models.Article{}).Where("status = ? AND \"updatedAt\" >= ?", models.ArticleStatusPublished, today).Count(&publishedTodayCount)

	var draftsCount int64
	h.DB.Model(&models.Article{}).Where("status = ?", models.ArticleStatusDraft).Count(&draftsCount)

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"articles":            articles,
		"total":               total,
		"publishedTodayCount": publishedTodayCount,
		"draftsCount":         draftsCount,
		"page":                page,
		"limit":               limit,
		"totalPages":          totalPages,
	})
}

func (h *AdminHandler) ChangeArticleStatus(w http.ResponseWriter, r *http.Request) {
	articleId := r.PathValue("articleId")
	if articleId == "" {
		response.Error(w, http.StatusBadRequest, "Article ID is required")
		return
	}

	var req struct {
		Status      string  `json:"status" validate:"required,oneof=DRAFT PUBLISHED ARCHIVED"`
		PublishedAt *string `json:"publishedAt,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid article status", "errors": errs})
		return
	}

	updates := map[string]any{"status": req.Status}
	switch req.Status {
	case "PUBLISHED":
		if req.PublishedAt != nil && *req.PublishedAt == "original" {
			var article models.Article
			h.DB.Select("\"createdAt\"").First(&article, "id = ?", articleId)
			updates["publishedAt"] = article.CreatedAt
		} else if req.PublishedAt != nil {
			t, err := time.Parse(time.RFC3339, *req.PublishedAt)
			if err == nil {
				updates["publishedAt"] = t
			} else {
				updates["publishedAt"] = time.Now()
			}
		} else {
			updates["publishedAt"] = time.Now()
		}
	case "DRAFT":
		updates["publishedAt"] = nil
	}

	result := h.DB.Model(&models.Article{}).Where("id = ?", articleId).Updates(updates)
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update article status")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	var article models.Article
	h.DB.First(&article, "id = ?", articleId)

	// Fire SEO notifications when article is published
	if req.Status == "PUBLISHED" {
		services.NotifyArticlePublished(h.Cfg.FrontendURL, article.Slug)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"message": "Article status updated successfully", "article": article,
	})
}

func (h *AdminHandler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	articleId := r.PathValue("articleId")
	if articleId == "" {
		response.Error(w, http.StatusBadRequest, "Article ID is required")
		return
	}

	result := h.DB.Unscoped().Where("id = ?", articleId).Delete(&models.Article{})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete article")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Article deleted successfully"})
}

func (h *AdminHandler) BulkDeleteArticles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.IDs) == 0 {
		response.Error(w, http.StatusBadRequest, "Invalid or empty article IDs")
		return
	}

	result := h.DB.Unscoped().Where("id IN ?", req.IDs).Delete(&models.Article{})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to bulk delete articles")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("%d articles deleted successfully", result.RowsAffected),
	})
}

func (h *AdminHandler) BulkChangeArticleStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs         []string `json:"ids"`
		Status      string   `json:"status"`
		PublishedAt *string  `json:"publishedAt,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.IDs) == 0 {
		response.Error(w, http.StatusBadRequest, "Invalid or empty article IDs")
		return
	}
	if req.Status != "DRAFT" && req.Status != "PUBLISHED" && req.Status != "ARCHIVED" {
		response.Error(w, http.StatusBadRequest, "Invalid article status")
		return
	}

	if req.Status == "PUBLISHED" && req.PublishedAt != nil && *req.PublishedAt == "original" {
		for _, id := range req.IDs {
			var article models.Article
			h.DB.Select("id", "\"createdAt\"").First(&article, "id = ?", id)
			h.DB.Model(&models.Article{}).Where("id = ?", id).Updates(map[string]any{
				"status": req.Status, "publishedAt": article.CreatedAt,
			})
		}
	} else {
		updates := map[string]any{"status": req.Status}
		switch req.Status {
		case "PUBLISHED":
			updates["publishedAt"] = time.Now()
		case "DRAFT":
			updates["publishedAt"] = nil
		}
		h.DB.Model(&models.Article{}).Where("id IN ?", req.IDs).Updates(updates)
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Articles updated successfully",
	})
}

func (h *AdminHandler) BulkDeleteUsers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.IDs) == 0 {
		response.Error(w, http.StatusBadRequest, "Invalid or empty user IDs")
		return
	}

	result := h.DB.Where("id IN ? AND role != ?", req.IDs, models.RoleAdmin).Delete(&models.User{})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to bulk delete users")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Users deleted successfully"})
}

func (h *AdminHandler) BulkUpdateUsers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs      []string `json:"ids"`
		Role     *string  `json:"role,omitempty"`
		IsActive *bool    `json:"isActive,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := map[string]any{}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.IsActive != nil {
		updates["isActive"] = *req.IsActive
	}

	if len(updates) == 0 || len(req.IDs) == 0 {
		response.Error(w, http.StatusBadRequest, "No updates or IDs provided")
		return
	}

	result := h.DB.Model(&models.User{}).Where("id IN ? AND role != ?", req.IDs, models.RoleAdmin).Updates(updates)
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to bulk update users")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Users updated successfully"})
}
