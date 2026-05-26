package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/middleware"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/services"
	"gorm.io/gorm"
)

type BookmarkHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewBookmarkHandler(db *gorm.DB, cfg *config.Config) *BookmarkHandler {
	return &BookmarkHandler{DB: db, Cfg: cfg}
}

type bookmarkRequest struct {
	ArticleID string `json:"articleId" validate:"required"`
}

func (h *BookmarkHandler) Add(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req bookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ArticleID == "" {
		response.Error(w, http.StatusBadRequest, "Article ID is required")
		return
	}

	bookmark := models.Bookmark{
		UserID:    user.UserID,
		ArticleID: req.ArticleID,
	}

	if err := h.DB.Create(&bookmark).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			response.JSON(w, http.StatusOK, map[string]any{
				"success": true, "message": "Article already bookmarked",
			})
			return
		}
		response.Error(w, http.StatusInternalServerError, "Failed to bookmark article")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"success": true, "message": "Article bookmarked successfully", "bookmark": bookmark,
	})
}

func (h *BookmarkHandler) Remove(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req bookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.ArticleID == "" {
		response.Error(w, http.StatusBadRequest, "Article ID is required")
		return
	}

	result := h.DB.Where("user_id = ? AND article_id = ?", user.UserID, req.ArticleID).Delete(&models.Bookmark{})
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to remove bookmark")
		return
	}
	if result.RowsAffected == 0 {
		response.JSON(w, http.StatusNotFound, map[string]any{
			"success": false, "message": "Bookmark not found",
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success": true, "message": "Bookmark removed successfully",
	})
}

func (h *BookmarkHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	articleId := r.PathValue("articleId")
	if articleId == "" {
		response.Error(w, http.StatusBadRequest, "Article ID is required")
		return
	}

	var bookmark models.Bookmark
	if err := h.DB.Where("user_id = ? AND article_id = ?", user.UserID, articleId).First(&bookmark).Error; err != nil {
		response.JSON(w, http.StatusNotFound, map[string]any{
			"success": false, "message": "Bookmark not found",
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success": true, "message": "Article is bookmarked", "bookmark": bookmark,
	})
}

func (h *BookmarkHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var bookmarks []models.Bookmark
	var total int64

	p, _ := services.NewPagination(r.URL.Query().Get("page"), r.URL.Query().Get("limit"))

	h.DB.Where("user_id = ?", user.UserID).
		Preload("Article", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Author", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "name", "email")
			})
		}).
		Offset(p.OffSet).Limit(p.Limit).
		Find(&bookmarks)
	h.DB.Model(&models.Bookmark{}).Where("user_id = ?", user.UserID).Count(&total)
	p.Total = total
	p.Calculate()

	if len(bookmarks) == 0 {
		bookmarks = []models.Bookmark{}
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success": true, "bookmarks": bookmarks, "bookmarkCount": total, "pagination": p,
	})
}
