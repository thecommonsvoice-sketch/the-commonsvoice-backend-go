package router

import (
	"net/http"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/handlers"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/middleware"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"gorm.io/gorm"
)

func Register(mux *http.ServeMux, db *gorm.DB, cfg *config.Config) {

	auth := handlers.NewAuthHandler(db, cfg)
	article := handlers.NewArticleHandler(db, cfg)
	category := handlers.NewCategoryHandler(db, cfg)
	bookmark := handlers.NewBookmarkHandler(db, cfg)
	comment := handlers.NewCommentHandler(db, cfg)
	admin := handlers.NewAdminHandler(db, cfg)
	news := handlers.NewNewsHandler(db, cfg)
	contact := handlers.NewContactHandler(cfg)

	// Health

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, 200, map[string]any{
			"status":    "ok",
			"timestamp": time.Now(),
		})
	})

	// Auth
	mux.HandleFunc("POST /api/auth/register", auth.Register)
	mux.HandleFunc("POST /api/auth/login", auth.Login)
	mux.HandleFunc("POST /api/auth/refresh", auth.Refresh)
	mux.HandleFunc("POST /api/auth/logout", auth.Logout)
	mux.HandleFunc("GET /api/auth/me", middleware.Authenticate(http.HandlerFunc(auth.Me), cfg.JWTSecret).ServeHTTP)

	// Article
	mux.Handle("GET /api/articles", middleware.CheckUser(http.HandlerFunc(article.List), cfg.JWTSecret))
	mux.HandleFunc("GET /api/articles/adjacent/{slug}", article.Adjacent)
	mux.HandleFunc("GET /api/articles/related/{slug}", article.Related)
	mux.HandleFunc("GET /api/articles/{slugOrId}", article.GetBySlugOrId)

	mux.Handle("POST /api/articles", middleware.Authenticate(
		middleware.Authorize("EDITOR", "REPORTER", "ADMIN")(
			http.HandlerFunc(article.Create),
		), cfg.JWTSecret,
	))

	mux.Handle("PUT /api/articles/{slugOrId}", middleware.Authenticate(
		middleware.Authorize("EDITOR", "REPORTER", "ADMIN")(
			http.HandlerFunc(article.Update),
		), cfg.JWTSecret,
	))

	mux.Handle("DELETE /api/articles/{slugOrId}", middleware.Authenticate(
		middleware.Authorize("EDITOR", "REPORTER", "ADMIN")(
			http.HandlerFunc(article.Delete),
		), cfg.JWTSecret,
	))

	mux.Handle("PATCH /api/articles/restore/{slugOrId}", middleware.Authenticate(
		middleware.Authorize("ADMIN", "EDITOR")(
			http.HandlerFunc(article.Restore),
		), cfg.JWTSecret,
	))

	mux.Handle("PATCH /api/articles/status/{id}", middleware.Authenticate(
		middleware.Authorize("ADMIN", "EDITOR")(
			http.HandlerFunc(article.UpdateStatus),
		), cfg.JWTSecret,
	))

	mux.Handle("GET /api/articles/role-check/{slugOrId}", middleware.Authenticate(
		middleware.Authorize("EDITOR", "REPORTER", "ADMIN")(
			http.HandlerFunc(article.RoleCheck),
		), cfg.JWTSecret,
	))

	mux.Handle("PUT /api/articles/role-check/{slugOrId}", middleware.Authenticate(
		middleware.Authorize("EDITOR", "REPORTER", "ADMIN")(
			http.HandlerFunc(article.Update),
		), cfg.JWTSecret,
	))

	mux.Handle("POST /api/articles/bulk-delete", middleware.Authenticate(
		middleware.Authorize("EDITOR", "REPORTER", "ADMIN")(
			http.HandlerFunc(article.BulkDelete),
		), cfg.JWTSecret,
	))

	mux.Handle("PATCH /api/articles/bulk-status", middleware.Authenticate(
		middleware.Authorize("ADMIN", "EDITOR")(
			http.HandlerFunc(article.BulkUpdateStatus),
		), cfg.JWTSecret,
	))

	// Category
	mux.HandleFunc("GET /api/categories", category.List)
	mux.HandleFunc("GET /api/categories/all-with-hierarchy", category.ListWithHierarchy)
	mux.Handle("GET /api/categories/inactive", middleware.Authenticate(
		middleware.Authorize("ADMIN")(
			http.HandlerFunc(category.ListInactive),
		), cfg.JWTSecret,
	))
	mux.HandleFunc("GET /api/categories/{slugOrId}", category.GetBySlugOrId)

	mux.Handle("POST /api/categories", middleware.Authenticate(
		middleware.Authorize("ADMIN", "EDITOR")(
			http.HandlerFunc(category.Create),
		), cfg.JWTSecret,
	))

	mux.Handle("PUT /api/categories/{slugOrId}", middleware.Authenticate(
		middleware.Authorize("ADMIN", "EDITOR")(
			http.HandlerFunc(category.Update),
		), cfg.JWTSecret,
	))

	mux.Handle("PATCH /api/categories/{slugOrId}/restore", middleware.Authenticate(
		middleware.Authorize("ADMIN")(
			http.HandlerFunc(category.Restore),
		), cfg.JWTSecret,
	))

	mux.Handle("DELETE /api/categories/{slugOrId}", middleware.Authenticate(
		middleware.Authorize("ADMIN", "EDITOR")(
			http.HandlerFunc(category.Delete),
		), cfg.JWTSecret,
	))

	mux.Handle("DELETE /api/categories/{id}/permanent", middleware.Authenticate(
		middleware.Authorize("ADMIN")(
			http.HandlerFunc(category.HardDelete),
		), cfg.JWTSecret,
	))

	// Bookmark
	mux.Handle("POST /api/bookmarks", middleware.Authenticate(
		http.HandlerFunc(bookmark.Add), cfg.JWTSecret,
	))
	mux.Handle("DELETE /api/bookmarks", middleware.Authenticate(
		http.HandlerFunc(bookmark.Remove), cfg.JWTSecret,
	))
	mux.Handle("GET /api/bookmarks", middleware.Authenticate(
		http.HandlerFunc(bookmark.List), cfg.JWTSecret,
	))
	mux.Handle("GET /api/bookmarks/{articleId}", middleware.Authenticate(
		http.HandlerFunc(bookmark.Get), cfg.JWTSecret,
	))

	// Comment
	mux.HandleFunc("GET /api/comments/{articleId}", comment.ListByArticle)

	mux.Handle("POST /api/comments", middleware.Authenticate(
		http.HandlerFunc(comment.Add), cfg.JWTSecret,
	))
	mux.Handle("GET /api/comments/user/{userId}", middleware.Authenticate(
		http.HandlerFunc(comment.ListByUser), cfg.JWTSecret,
	))
	mux.Handle("DELETE /api/comments/{commentId}", middleware.Authenticate(
		http.HandlerFunc(comment.Delete), cfg.JWTSecret,
	))
	mux.Handle("PUT /api/comments/{commentId}", middleware.Authenticate(
		http.HandlerFunc(comment.Edit), cfg.JWTSecret,
	))
	mux.Handle("PUT /api/comments/{commentId}/reply", middleware.Authenticate(
		http.HandlerFunc(comment.Reply), cfg.JWTSecret,
	))

	// Admin (all routes require ADMIN role)
	adminAuth := func(h http.Handler) http.Handler {
		return middleware.Authenticate(middleware.Authorize("ADMIN")(h), cfg.JWTSecret)
	}

	mux.Handle("GET /api/admin/users", adminAuth(http.HandlerFunc(admin.GetAllUsers)))
	mux.Handle("POST /api/admin/users", adminAuth(http.HandlerFunc(admin.CreateUser)))
	mux.Handle("PATCH /api/admin/users/bulk-update", adminAuth(http.HandlerFunc(admin.BulkUpdateUsers)))
	mux.Handle("POST /api/admin/users/bulk-delete", adminAuth(http.HandlerFunc(admin.BulkDeleteUsers)))
	mux.Handle("PATCH /api/admin/users/{userId}/role", adminAuth(http.HandlerFunc(admin.UpdateUserRole)))
	mux.Handle("PATCH /api/admin/users/{userId}/toggle", adminAuth(http.HandlerFunc(admin.ToggleUserActiveStatus)))

	mux.Handle("GET /api/admin/articles", adminAuth(http.HandlerFunc(admin.GetAllArticles)))
	mux.Handle("POST /api/admin/articles/bulk-delete", adminAuth(http.HandlerFunc(admin.BulkDeleteArticles)))
	mux.Handle("PATCH /api/admin/articles/bulk-status", adminAuth(http.HandlerFunc(admin.BulkChangeArticleStatus)))
	mux.Handle("GET /api/admin/articles/{slugOrId}", adminAuth(http.HandlerFunc(admin.GetArticleBySlugOrId)))
	mux.Handle("PATCH /api/admin/articles/{articleId}/status", adminAuth(http.HandlerFunc(admin.ChangeArticleStatus)))
	mux.Handle("DELETE /api/admin/articles/{articleId}", adminAuth(http.HandlerFunc(admin.DeleteArticle)))

	// News (public read, cron-protected write)
	cron := func(h http.Handler) http.Handler {
		return middleware.CronAuth(h, cfg.CronSecret)
	}

	mux.Handle("GET /api/fetch-latest-news", cron(http.HandlerFunc(news.FetchLatestNews)))
	mux.Handle("GET /api/fetch-news-cat", cron(http.HandlerFunc(news.FetchNewsByCategory)))
	mux.Handle("GET /api/cleanup-news", cron(http.HandlerFunc(news.CleanupOldNews)))

	mux.HandleFunc("GET /api/news", news.GetCachedNews)
	mux.HandleFunc("GET /api/news/business", news.FetchBusinessNews)
	mux.HandleFunc("GET /api/news/sports", news.FetchSportsNews)
	mux.HandleFunc("GET /api/news/tech", news.FetchTechNews)
	mux.HandleFunc("GET /api/news/science", news.FetchScienceNews)
	mux.HandleFunc("GET /api/news/health", news.FetchHealthNews)
	mux.HandleFunc("GET /api/news/entertainment", news.FetchEntertainmentNews)
	mux.HandleFunc("GET /api/news/fashion", news.FetchFashionNews)

	// Contact
	mux.HandleFunc("POST /api/contact/send", contact.SendContactMail)

}
