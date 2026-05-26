package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/middleware"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/services"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/validator"
	"gorm.io/gorm"
)

type ArticleHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewArticleHandler(db *gorm.DB, cfg *config.Config) *ArticleHandler {
	return &ArticleHandler{DB: db, Cfg: cfg}
}

type createArticleRequest struct {
	Title           string   `json:"title" validate:"required,min=3"`
	Slug            string   `json:"slug,omitempty"`
	Content         string   `json:"content" validate:"required,min=10"`
	CategoryId      string   `json:"categoryId" validate:"required"`
	Tags            []string `json:"tags,omitempty"`
	Status          string   `json:"status,omitempty"`
	CoverImage      string   `json:"coverImage,omitempty"`
	MetaTitle       string   `json:"metaTitle,omitempty"`
	MetaDescription string   `json:"metaDescription,omitempty"`
	OgImage         string   `json:"ogImage,omitempty"`
	Excerpt         string   `json:"excerpt,omitempty"`
	VideoTitle      string   `json:"videoTitle,omitempty"`
	VideoURL        string   `json:"videoUrl,omitempty"`
	VideoType       string   `json:"videoType,omitempty"`
}

type updateArticleRequest struct {
	Title           *string   `json:"title,omitempty" validate:"omitempty,min=3"`
	Content         *string   `json:"content,omitempty" validate:"omitempty,min=10"`
	CategoryId      *string   `json:"categoryId,omitempty"`
	Tags            []string  `json:"tags,omitempty"`
	Status          *string   `json:"status,omitempty"`
	CoverImage      *string   `json:"coverImage,omitempty" validate:"omitempty,url"`
	MetaTitle       *string   `json:"metaTitle,omitempty" validate:"omitempty,max=60"`
	MetaDescription *string   `json:"metaDescription,omitempty" validate:"omitempty,max=160"`
	OgImage         *string   `json:"ogImage,omitempty"`
	Excerpt         *string   `json:"excerpt,omitempty"`
	VideoTitle      *string   `json:"videoTitle,omitempty"`
	VideoURL        *string   `json:"videoUrl,omitempty"`
	VideoType       *string   `json:"videoType,omitempty"`
}

func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var req createArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid article data", "errors": errs})
		return
	}

	slug := req.Slug
	if slug == "" {
		var err error
		slug, err = services.CreateSlug(req.Title, "", h.DB)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to generate slug")
			return
		}
	}

	categoryId := req.CategoryId

	metaTitle := req.MetaTitle
	if metaTitle == "" {
		if len(req.Title) > 60 {
			metaTitle = req.Title[:60]
		} else {
			metaTitle = req.Title
		}
	}

	metaDescription := req.MetaDescription
	if metaDescription == "" {
		if len(req.Content) > 160 {
			metaDescription = req.Content[:160]
		} else {
			metaDescription = req.Content
		}
	}

	article := models.Article{
		Title:           req.Title,
		Slug:            slug,
		Content:         req.Content,
		CategoryID:      categoryId,
		AuthorID:        user.UserID,
		Tags:            models.StringArray(req.Tags),
		CoverImage:      ifPtr(req.CoverImage),
		MetaTitle:       &metaTitle,
		MetaDescription: &metaDescription,
		OgImage:         ifPtr(req.OgImage),
		Excerpt:         ifPtr(req.Excerpt),
	}

	if req.Status == "PUBLISHED" {
		article.Status = models.ArticleStatusPublished
		now := time.Now()
		article.PublishedAt = &now
	} else if req.Status == "ARCHIVED" {
		article.Status = models.ArticleStatusArchived
	} else {
		article.Status = models.ArticleStatusDraft
	}

	if err := h.DB.Create(&article).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create article")
		return
	}

	if req.VideoURL != "" {
		video := models.ArticleVideo{
			ArticleID: article.ID,
			Type:      req.VideoType,
			URL:       req.VideoURL,
			Title:     ifPtr(req.VideoTitle),
		}
		h.DB.Create(&video)
	}

	var created models.Article
	h.DB.Preload("Author", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "email")
	}).Preload("Category", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "slug")
	}).Preload("Videos").First(&created, "id = ?", article.ID)

	response.JSON(w, http.StatusCreated, map[string]any{"message": "Article created successfully", "article": created})
}

func ifPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *ArticleHandler) GetBySlugOrId(w http.ResponseWriter, r *http.Request) {
	slugOrId := r.PathValue("slugOrId")

	var article models.Article
	if err := h.DB.Where("\"deletedAt\" IS NULL").Preload("Author", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "email")
	}).Preload("Category", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "slug")
	}).Preload("Videos").Where("id = ? OR slug = ?", slugOrId, slugOrId).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"article": article})
}

func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	slugOrId := r.PathValue("slugOrId")

	var existing models.Article
	if err := h.DB.Where("id = ? OR slug = ?", slugOrId, slugOrId).First(&existing).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	if existing.DeletedAt != nil {
		response.Error(w, http.StatusBadRequest, "Article is deleted")
		return
	}

	if user.Role != "ADMIN" && user.Role != "EDITOR" && existing.AuthorID != user.UserID {
		response.Error(w, http.StatusForbidden, "You can only edit your own articles")
		return
	}

	var req updateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		for field := range errs {
			if field == "title" || field == "content" || field == "coverImage" || field == "metaTitle" || field == "metaDescription" {
				response.JSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid article data", "errors": errs})
				return
			}
		}
	}

	if req.Title != nil {
		existing.Title = *req.Title
		newSlug, err := services.CreateSlug(*req.Title, existing.ID, h.DB)
		if err == nil {
			existing.Slug = newSlug
		}
	}
	if req.Content != nil {
		existing.Content = *req.Content
	}
	if req.CategoryId != nil {
		existing.CategoryID = *req.CategoryId
	}
	if req.CoverImage != nil {
		if *req.CoverImage == "" {
			existing.CoverImage = nil
		} else {
			existing.CoverImage = req.CoverImage
		}
	}
	if req.MetaTitle != nil {
		existing.MetaTitle = req.MetaTitle
	}
	if req.MetaDescription != nil {
		existing.MetaDescription = req.MetaDescription
	}
	if req.OgImage != nil {
		existing.OgImage = req.OgImage
	}
	if req.Tags != nil {
		existing.Tags = models.StringArray(req.Tags)
	}
	if req.Excerpt != nil {
		existing.Excerpt = req.Excerpt
	}

	// Handle video replacement
	if req.VideoURL != nil {
		h.DB.Where("\"articleId\" = ?", existing.ID).Delete(&models.ArticleVideo{})
		if *req.VideoURL != "" {
			video := models.ArticleVideo{
				ArticleID: existing.ID,
				Type:      *req.VideoType,
				URL:       *req.VideoURL,
				Title:     req.VideoTitle,
			}
			h.DB.Create(&video)
		}
	}

	if err := h.DB.Save(&existing).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update article")
		return
	}

	var updated models.Article
	h.DB.Preload("Author", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "email")
	}).Preload("Category", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "slug")
	}).Preload("Videos").First(&updated, "id = ?", existing.ID)

	response.JSON(w, http.StatusOK, map[string]any{"message": "Article updated successfully", "article": updated})
}

func (h *ArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	slugOrId := r.PathValue("slugOrId")

	var article models.Article
	if err := h.DB.Unscoped().Where("id = ? OR slug = ?", slugOrId, slugOrId).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	if article.DeletedAt != nil {
		if err := h.DB.Unscoped().Delete(&article).Error; err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to permanently delete article")
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"message": "Article permanently deleted"})
		return
	}

	if user.Role == "REPORTER" && article.AuthorID != user.UserID {
		response.Error(w, http.StatusForbidden, "You can only delete your own articles")
		return
	}

	now := time.Now()
	article.DeletedAt = &now
	if err := h.DB.Save(&article).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete article")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Article deleted successfully"})
}

func (h *ArticleHandler) Restore(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	slugOrId := r.PathValue("slugOrId")

	var article models.Article
	if err := h.DB.Unscoped().Where("id = ? OR slug = ?", slugOrId, slugOrId).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	if article.DeletedAt == nil {
		response.Error(w, http.StatusBadRequest, "Article is not deleted")
		return
	}

	article.DeletedAt = nil
	if err := h.DB.Save(&article).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to restore article")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Article restored successfully"})
}

type updateStatusRequest struct {
	Status      string  `json:"status" validate:"required,oneof=DRAFT PUBLISHED ARCHIVED"`
	PublishedAt *string `json:"publishedAt,omitempty"`
}

func (h *ArticleHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	id := r.PathValue("id")

	var article models.Article
	if err := h.DB.Where("id = ?", id).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid article status", "errors": errs})
		return
	}

	article.Status = models.ArticleStatus(req.Status)

	if req.PublishedAt != nil && *req.PublishedAt == "original" {
		article.PublishedAt = &article.CreatedAt
	} else if req.Status == "PUBLISHED" {
		now := time.Now()
		article.PublishedAt = &now
	} else if req.Status == "DRAFT" {
		article.PublishedAt = nil
	}

	if err := h.DB.Save(&article).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update article status")
		return
	}

	// Fire SEO notifications when article is published
	if req.Status == "PUBLISHED" {
		services.NotifyArticlePublished(h.Cfg.FrontendURL, article.Slug)
	}

	response.JSON(w, http.StatusOK, map[string]any{"message": "Article status updated successfully", "article": article})
}

func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	p, _ := services.NewPagination(r.URL.Query().Get("page"), r.URL.Query().Get("limit"))

	filter := func(db *gorm.DB) *gorm.DB {
		db = db.Where("\"deletedAt\" IS NULL")

		statusFilter := r.URL.Query().Get("status")
		if user == nil {
			db = db.Where("status = ?", models.ArticleStatusPublished)
		} else if statusFilter != "" {
			db = db.Where("status = ?", statusFilter)
		}

		if search := r.URL.Query().Get("search"); search != "" {
			like := "%" + search + "%"
			db = db.Where("(title ILIKE ? OR content ILIKE ?)", like, like)
		}

		if author := r.URL.Query().Get("author"); author != "" {
			db = db.Where("\"authorId\" IN (SELECT id FROM \"User\" WHERE name ILIKE ?)", "%"+author+"%")
		}

		if authorId := r.URL.Query().Get("authorId"); authorId != "" {
			db = db.Where("\"authorId\" = ?", authorId)
		}

		if startDate := r.URL.Query().Get("startDate"); startDate != "" {
			if t, err := time.Parse("2006-01-02", startDate); err == nil {
				db = db.Where("\"createdAt\" >= ?", t)
			}
		}
		if endDate := r.URL.Query().Get("endDate"); endDate != "" {
			if t, err := time.Parse("2006-01-02", endDate); err == nil {
				db = db.Where("\"createdAt\" <= ?", t.Add(24*time.Hour))
			}
		}

		if category := r.URL.Query().Get("category"); category != "" {
			var cat models.Category
			if err := h.DB.Where("slug = ? OR name = ?", category, category).First(&cat).Error; err == nil {
				var childIDs []string
				h.DB.Model(&models.Category{}).
					Where("id = ? OR \"parentId\" = ?", cat.ID, cat.ID).
					Pluck("id", &childIDs)
				db = db.Where("\"categoryId\" IN ?", childIDs)
			}
		}

		return db
	}

	var total int64
	filter(h.DB.Model(&models.Article{})).Count(&total)
	p.Total = total
	p.Calculate()

	var articles []models.Article
	filter(h.DB.Model(&models.Article{})).
		Select("id", "title", "slug", "content", "excerpt", "\"coverImage\"", "\"ogImage\"", "status", "\"metaTitle\"", "\"metaDescription\"", "tags", "\"authorId\"", "\"categoryId\"", "\"createdAt\"", "\"updatedAt\"", "\"publishedAt\"").
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name")
		}).
		Preload("Category", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "slug")
		}).
		Offset(p.OffSet).Limit(p.Limit).
		Order("\"createdAt\" DESC").
		Find(&articles)

	today := time.Now().Truncate(24 * time.Hour)
	var updatedTodayCount int64
	filter(h.DB.Model(&models.Article{})).Where("\"updatedAt\" >= ?", today).Count(&updatedTodayCount)

	var draftCount int64
	filter(h.DB.Model(&models.Article{})).Where("status = ?", "DRAFT").Count(&draftCount)

	bookmarkSet := make(map[string]bool)
	if user != nil && len(articles) > 0 {
		var articleIDs []string
		for _, a := range articles {
			articleIDs = append(articleIDs, a.ID)
		}
		var bookmarks []models.Bookmark
		h.DB.Where("\"userId\" = ? AND \"articleId\" IN ?", user.UserID, articleIDs).Find(&bookmarks)
		for _, b := range bookmarks {
			bookmarkSet[b.ArticleID] = true
		}
	}

	type articleItem struct {
		ID              string              `json:"id"`
		Title           string              `json:"title"`
		Slug            string              `json:"slug"`
		Content         string              `json:"content"`
		Excerpt         *string             `json:"excerpt"`
		CoverImage      *string             `json:"coverImage"`
		OgImage         *string             `json:"ogImage"`
		Status          models.ArticleStatus `json:"status"`
		MetaTitle       *string             `json:"metaTitle"`
		MetaDescription *string             `json:"metaDescription"`
		Tags            []string            `json:"tags"`
		Category        *models.Category    `json:"category"`
		Author          *models.User        `json:"author"`
		CreatedAt       time.Time           `json:"createdAt"`
		UpdatedAt       time.Time           `json:"updatedAt"`
		PublishedAt     *time.Time          `json:"publishedAt"`
		IsBookmarked    bool                `json:"isBookmarked"`
	}

	data := make([]articleItem, 0, len(articles))
	for _, a := range articles {
		data = append(data, articleItem{
			ID:              a.ID,
			Title:           a.Title,
			Slug:            a.Slug,
			Content:         a.Content,
			Excerpt:         a.Excerpt,
			CoverImage:      a.CoverImage,
			OgImage:         a.OgImage,
			Status:          a.Status,
			MetaTitle:       a.MetaTitle,
			MetaDescription: a.MetaDescription,
			Tags:            a.Tags,
			Category:        &a.Category,
			Author:          &a.Author,
			CreatedAt:       a.CreatedAt,
			UpdatedAt:       a.UpdatedAt,
			PublishedAt:     a.PublishedAt,
			IsBookmarked:    bookmarkSet[a.ID],
		})
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data":               data,
		"pagination":         p,
		"updatedTodayCount":  updatedTodayCount,
		"draftCount":         draftCount,
	})
}

func (h *ArticleHandler) Adjacent(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		response.Error(w, http.StatusBadRequest, "Slug is required")
		return
	}

	var article models.Article
	if err := h.DB.Where("slug = ?", slug).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	var next, prev *struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}

	var nextArticle models.Article
	if err := h.DB.Where("\"createdAt\" > ? AND status = ? AND \"deletedAt\" IS NULL", article.CreatedAt, models.ArticleStatusPublished).
		Order("\"createdAt\" ASC").Limit(1).First(&nextArticle).Error; err == nil {
		next = &struct {
			Slug  string `json:"slug"`
			Title string `json:"title"`
		}{Slug: nextArticle.Slug, Title: nextArticle.Title}
	}

	var prevArticle models.Article
	if err := h.DB.Where("\"createdAt\" < ? AND status = ? AND \"deletedAt\" IS NULL", article.CreatedAt, models.ArticleStatusPublished).
		Order("\"createdAt\" DESC").Limit(1).First(&prevArticle).Error; err == nil {
		prev = &struct {
			Slug  string `json:"slug"`
			Title string `json:"title"`
		}{Slug: prevArticle.Slug, Title: prevArticle.Title}
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"next": next,
		"prev": prev,
	})
}

func (h *ArticleHandler) Related(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		response.Error(w, http.StatusBadRequest, "Slug is required")
		return
	}

	var article models.Article
	if err := h.DB.Where("slug = ?", slug).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	const limit = 6
	result := make([]models.Article, 0, limit)
	usedIDs := map[string]bool{article.ID: true}

	// Step 1: By shared tags
	if len(article.Tags) > 0 {
		conditions := make([]string, 0, len(article.Tags))
		args := make([]any, 0, len(article.Tags)+3)
		args = append(args, article.ID, models.ArticleStatusPublished)
		for _, tag := range article.Tags {
			conditions = append(conditions, "? = ANY(tags)")
			args = append(args, tag)
		}
		where := fmt.Sprintf("id != ? AND status = ? AND \"deletedAt\" IS NULL AND (%s)", strings.Join(conditions, " OR "))

		var tagArticles []models.Article
		h.DB.Where(where, args...).
			Select("id", "title", "slug", "\"coverImage\"", "excerpt", "tags", "\"categoryId\"", "\"authorId\"", "\"createdAt\"", "\"publishedAt\"").
			Preload("Author", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "name")
			}).
			Order("\"createdAt\" DESC").
			Limit(limit).
			Find(&tagArticles)

		for _, a := range tagArticles {
			if len(result) >= limit {
				break
			}
			if !usedIDs[a.ID] {
				usedIDs[a.ID] = true
				result = append(result, a)
			}
		}
	}

	// Step 2: By same category
	if len(result) < limit {
		var catArticles []models.Article
		h.DB.Where("\"categoryId\" = ? AND id != ? AND status = ? AND \"deletedAt\" IS NULL", article.CategoryID, article.ID, models.ArticleStatusPublished).
			Select("id", "title", "slug", "\"coverImage\"", "excerpt", "tags", "\"categoryId\"", "\"authorId\"", "\"createdAt\"", "\"publishedAt\"").
			Preload("Author", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "name")
			}).
			Order("\"createdAt\" DESC").
			Limit(limit - len(result)).
			Find(&catArticles)

		for _, a := range catArticles {
			if len(result) >= limit {
				break
			}
			if !usedIDs[a.ID] {
				usedIDs[a.ID] = true
				result = append(result, a)
			}
		}
	}

	// Step 3: Recent articles as fallback
	if len(result) < limit {
		var recentArticles []models.Article
		h.DB.Where("id != ? AND status = ? AND \"deletedAt\" IS NULL", article.ID, models.ArticleStatusPublished).
			Select("id", "title", "slug", "\"coverImage\"", "excerpt", "tags", "\"categoryId\"", "\"authorId\"", "\"createdAt\"", "\"publishedAt\"").
			Preload("Author", func(db *gorm.DB) *gorm.DB {
				return db.Select("id", "name")
			}).
			Order("\"createdAt\" DESC").
			Limit(limit - len(result)).
			Find(&recentArticles)

		for _, a := range recentArticles {
			if len(result) >= limit {
				break
			}
			if !usedIDs[a.ID] {
				usedIDs[a.ID] = true
				result = append(result, a)
			}
		}
	}

	response.JSON(w, http.StatusOK, map[string]any{"articles": result})
}

func (h *ArticleHandler) RoleCheck(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	slugOrId := r.PathValue("slugOrId")

	var article models.Article
	if err := h.DB.Unscoped().Where("id = ? OR slug = ?", slugOrId, slugOrId).First(&article).Error; err != nil {
		response.Error(w, http.StatusNotFound, "Article not found")
		return
	}

	if article.DeletedAt != nil {
		response.Error(w, http.StatusBadRequest, "Article is deleted")
		return
	}

	if user.Role == "REPORTER" && article.AuthorID != user.UserID {
		response.Error(w, http.StatusForbidden, "You are not the author of this article")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"allowed": true})
}

func (h *ArticleHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Error(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

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

	if user.Role == "REPORTER" {
		var count int64
		h.DB.Model(&models.Article{}).Where("id IN ? AND \"authorId\" != ?", req.IDs, user.UserID).Count(&count)
		if count > 0 {
			response.Error(w, http.StatusForbidden, "You can only delete your own articles")
			return
		}
	}

	now := time.Now()
	result := h.DB.Model(&models.Article{}).Where("id IN ?", req.IDs).Update("\"deletedAt\"", now)
	if result.Error != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to bulk delete articles")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("%d articles deleted successfully", result.RowsAffected),
	})
}

type bulkStatusRequest struct {
	IDs         []string `json:"ids"`
	Status      string   `json:"status"`
	PublishedAt *string  `json:"publishedAt,omitempty"`
}

func (h *ArticleHandler) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req bulkStatusRequest
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
		h.DB.Model(&models.Article{}).Where("id IN ?", req.IDs).
			Update("\"publishedAt\"", gorm.Expr("\"createdAt\""))
	} else {
		updates := map[string]any{"status": req.Status}
		if req.Status == "PUBLISHED" {
			publishedAt := time.Now()
			updates["publishedAt"] = publishedAt
		} else if req.Status == "DRAFT" {
			updates["publishedAt"] = nil
		}
		h.DB.Model(&models.Article{}).Where("id IN ?", req.IDs).Updates(updates)
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Articles updated successfully",
	})
}
