package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NewsHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewNewsHandler(db *gorm.DB, cfg *config.Config) *NewsHandler {
	return &NewsHandler{DB: db, Cfg: cfg}
}

func (h *NewsHandler) FetchLatestNews(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0

	wg.Add(2)

	go func() {
		defer wg.Done()
		articles, err := h.fetchNewsData()
		if err != nil {
			slog.Error("newsdata.io fetch failed", "error", err)
			return
		}
		mu.Lock()
		total += articles
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		articles, err := h.fetchTheNewsAPI()
		if err != nil {
			slog.Error("thenewsapi fetch failed", "error", err)
			return
		}
		mu.Lock()
		total += articles
		mu.Unlock()
	}()

	wg.Wait()

	response.JSON(w, http.StatusOK, map[string]any{
		"success": true, "message": "News updated successfully.", "count": total,
	})
}

func (h *NewsHandler) fetchNewsData() (int, error) {
	url := fmt.Sprintf("https://newsdata.io/api/1/latest?apikey=%s&country=in&language=en", h.Cfg.NewsDataAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Results []struct {
			ArticleID   string  `json:"article_id"`
			Title       string  `json:"title"`
			Description *string `json:"description"`
			Link        *string `json:"link"`
			ImageURL    *string `json:"image_url"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}

	count := 0
	for _, article := range apiResp.Results {
		if article.ArticleID == "" || article.Title == "" {
			continue
		}
		news := models.LatestNews{
			ID:          article.ArticleID,
			Title:       article.Title,
			PhotoURL:    article.ImageURL,
			Link:        article.Link,
			Description: article.Description,
		}
		h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&news)
		count++
	}
	return count, nil
}

func (h *NewsHandler) fetchTheNewsAPI() (int, error) {
	url := fmt.Sprintf("https://api.thenewsapi.com/v1/news/all?api_token=%s&language=en&locale=in&limit=20", h.Cfg.TheNewsAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Data []struct {
			UUID        string  `json:"uuid"`
			Title       string  `json:"title"`
			Description *string `json:"description"`
			URL         *string `json:"url"`
			ImageURL    *string `json:"image_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}

	count := 0
	for _, article := range apiResp.Data {
		if article.UUID == "" || article.Title == "" {
			continue
		}
		news := models.LatestNews{
			ID:          article.UUID,
			Title:       article.Title,
			PhotoURL:    article.ImageURL,
			Link:        article.URL,
			Description: article.Description,
		}
		h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&news)
		count++
	}
	return count, nil
}

func (h *NewsHandler) GetCachedNews(w http.ResponseWriter, r *http.Request) {
	var news []models.LatestNews
	result := h.DB.Order("\"createdAt\" DESC").Limit(50).Find(&news)
	if result.Error != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{
			"success": false, "error": result.Error.Error(),
		})
		return
	}
	if news == nil {
		news = []models.LatestNews{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

func (h *NewsHandler) fetchNewsByCategory(category string) (int, error) {
	baseURL := fmt.Sprintf("https://api.thenewsapi.com/v1/news/all?api_token=%s&language=en&locale=in&limit=20", h.Cfg.TheNewsAPIKey)

	urls := []string{
		baseURL + "&categories=" + category,
		baseURL + "&search=" + category,
		"https://api.thenewsapi.com/v1/news/all?api_token=" + h.Cfg.TheNewsAPIKey + "&language=en&categories=" + category + "&limit=20",
	}

	var articles []struct {
		UUID        string  `json:"uuid"`
		Title       string  `json:"title"`
		Description *string `json:"description"`
		URL         *string `json:"url"`
		ImageURL    *string `json:"image_url"`
	}

	client := &http.Client{Timeout: 15 * time.Second}

	for attempt, fetchURL := range urls {
		for retry := 0; retry < 3; retry++ {
			resp, err := client.Get(fetchURL)
			if err != nil {
				time.Sleep(time.Duration(math.Pow(2, float64(retry))) * time.Second)
				continue
			}

			var apiResp struct {
				Data []struct {
					UUID        string  `json:"uuid"`
					Title       string  `json:"title"`
					Description *string `json:"description"`
					URL         *string `json:"url"`
					ImageURL    *string `json:"image_url"`
				} `json:"data"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				resp.Body.Close()
				time.Sleep(time.Duration(math.Pow(2, float64(retry))) * time.Second)
				continue
			}
			resp.Body.Close()

			if len(apiResp.Data) > 0 {
				articles = apiResp.Data
				break
			}
			break
		}
		if len(articles) > 0 {
			break
		}
		if attempt == 2 {
			// last attempt already tried above
		}
	}

	if len(articles) == 0 {
		return 0, nil
	}

	count := 0
	for _, article := range articles {
		if article.UUID == "" || article.Title == "" {
			continue
		}
		typeStr := category
		news := models.LatestNews{
			ID:          article.UUID,
			Title:       article.Title,
			PhotoURL:    article.ImageURL,
			Link:        article.URL,
			Type:        &typeStr,
			Description: article.Description,
		}
		h.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&news)
		count++
	}

	return count, nil
}

func (h *NewsHandler) FetchNewsByCategory(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category == "" {
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"success": false, "error": "Category is required",
		})
		return
	}

	count, err := h.fetchNewsByCategory(category)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{
			"success": false, "error": err.Error(),
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"success": true, "count": count,
	})
}

func (h *NewsHandler) fetchNewsByType(w http.ResponseWriter, newsType string) {
	var news []models.LatestNews
	result := h.DB.Where("type = ?", newsType).Order("\"createdAt\" DESC").Limit(50).Find(&news)
	if result.Error != nil {
		response.JSON(w, http.StatusInternalServerError, map[string]any{
			"success": false, "error": result.Error.Error(),
		})
		return
	}
	if news == nil {
		news = []models.LatestNews{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

func (h *NewsHandler) FetchBusinessNews(w http.ResponseWriter, r *http.Request) {
	h.fetchNewsByType(w, "business")
}

func (h *NewsHandler) FetchSportsNews(w http.ResponseWriter, r *http.Request) {
	h.fetchNewsByType(w, "sports")
}

func (h *NewsHandler) FetchTechNews(w http.ResponseWriter, r *http.Request) {
	h.fetchNewsByType(w, "tech")
}

func (h *NewsHandler) FetchScienceNews(w http.ResponseWriter, r *http.Request) {
	h.fetchNewsByType(w, "science")
}

func (h *NewsHandler) FetchHealthNews(w http.ResponseWriter, r *http.Request) {
	h.fetchNewsByType(w, "health")
}

func (h *NewsHandler) FetchEntertainmentNews(w http.ResponseWriter, r *http.Request) {
	h.fetchNewsByType(w, "entertainment")
}

func (h *NewsHandler) FetchFashionNews(w http.ResponseWriter, r *http.Request) {
	h.fetchNewsByType(w, "fashion")
}

func (h *NewsHandler) CleanupOldNews(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 7
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	result := h.DB.Where("\"createdAt\" < ?", cutoff).Delete(&models.LatestNews{})

	response.JSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"deletedCount": result.RowsAffected,
		"message":      fmt.Sprintf("Deleted %d news records older than %d days.", result.RowsAffected, days),
	})
}
