package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/middleware"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"gorm.io/gorm"
)

type CommentHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewCommentHandler(db *gorm.DB, cfg *config.Config) *CommentHandler {
	return &CommentHandler{DB: db, Cfg: cfg}
}

type commentRequest struct {
	ArticleID string `json:"articleId"`
	Content   string `json:"content"`
}

func (h *CommentHandler) Add(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req commentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ArticleID == "" || strings.TrimSpace(req.Content) == "" {
		response.Error(w, http.StatusBadRequest, "Article ID and content are required")
		return
	}

	comment := models.Comment{
		Content:   req.Content,
		UserID:    user.UserID,
		ArticleID: req.ArticleID,
	}

	if err := h.DB.Create(&comment).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to add comment")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"message": "Comment added successfully", "comment": comment,
	})
}

func (h *CommentHandler) ListByArticle(w http.ResponseWriter, r *http.Request) {
	articleId := r.PathValue("articleId")
	if articleId == "" {
		response.Error(w, http.StatusBadRequest, "Article ID is required")
		return
	}

	var comments []models.Comment
	h.DB.Where("\"articleId\" = ?", articleId).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "email")
		}).
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Preload("User", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "name", "email")
			})
		}).
		Order("\"createdAt\" DESC").
		Find(&comments)

	if len(comments) == 0 {
		comments = []models.Comment{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (h *CommentHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("userId")
	if userId == "" {
		response.Error(w, http.StatusBadRequest, "User ID is required")
		return
	}

	var comments []models.Comment
	h.DB.Where("\"userId\" = ?", userId).
		Preload("Article", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "title", "slug")
		}).
		Order("\"createdAt\" DESC").
		Find(&comments)

	if len(comments) == 0 {
		comments = []models.Comment{}
	}

	response.JSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	commentId := r.PathValue("commentId")
	if commentId == "" {
		response.Error(w, http.StatusBadRequest, "Comment ID is required")
		return
	}

	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var comment models.Comment
	if err := h.DB.First(&comment, "id = ?", commentId).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Comment not found")
		return
	}

	if comment.UserID != user.UserID {
		response.Error(w, http.StatusForbidden, "You are not authorized to delete this comment")
		return
	}

	if err := h.DB.Delete(&comment).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete comment")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Comment deleted successfully"})
}

func (h *CommentHandler) Edit(w http.ResponseWriter, r *http.Request) {
	commentId := r.PathValue("commentId")
	if commentId == "" {
		response.Error(w, http.StatusBadRequest, "Comment ID is required")
		return
	}

	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		response.Error(w, http.StatusBadRequest, "Content is required")
		return
	}

	var comment models.Comment
	if err := h.DB.First(&comment, "id = ?", commentId).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Comment not found")
		return
	}

	if comment.UserID != user.UserID {
		response.Error(w, http.StatusForbidden, "You are not authorized to edit this comment")
		return
	}

	comment.Content = req.Content
	if err := h.DB.Save(&comment).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to edit comment")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"message": "Comment updated successfully", "updatedComment": comment,
	})
}

func (h *CommentHandler) Reply(w http.ResponseWriter, r *http.Request) {
	commentId := r.PathValue("commentId")
	if commentId == "" {
		response.Error(w, http.StatusBadRequest, "Comment ID is required")
		return
	}

	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		response.Error(w, http.StatusBadRequest, "Reply content cannot be empty")
		return
	}

	var parentComment models.Comment
	if err := h.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name")
	}).First(&parentComment, "id = ?", commentId).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Parent comment not found")
		return
	}

	if parentComment.UserID == user.UserID {
		response.Error(w, http.StatusBadRequest, "You cannot reply to your own comment")
		return
	}

	reply := models.Comment{
		Content:   req.Content,
		UserID:    user.UserID,
		ArticleID: parentComment.ArticleID,
		ParentID:  &commentId,
	}

	if err := h.DB.Create(&reply).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to add reply to comment")
		return
	}

	h.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "email")
	}).Preload("Parent", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "content", "\"userId\"")
	}).Preload("Parent.User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name")
	}).First(&reply, reply.ID)

	response.JSON(w, http.StatusCreated, map[string]any{
		"message": "Reply added successfully", "reply": reply,
	})
}
