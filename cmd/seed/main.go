package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/database"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type categorySeed struct {
	ID   string
	Name string
	Slug string
}

var categories = []categorySeed{
	{ID: "cm0a1b2c3d4e5f6a7b8c9001", Name: "General", Slug: "general"},
	{ID: "cm0a1b2c3d4e5f6a7b8c9002", Name: "Politics", Slug: "politics"},
	{ID: "cm0a1b2c3d4e5f6a7b8c9003", Name: "Science & Technology", Slug: "science-and-technology"},
	{ID: "cm0a1b2c3d4e5f6a7b8c9004", Name: "Sports & Entertainment", Slug: "sports-and-entertainment"},
	{ID: "cm0a1b2c3d4e5f6a7b8c9005", Name: "Business", Slug: "business"},
}

type articleSeed struct {
	Title      string
	Content    string
	CategoryID string
	Tags       []string
}

var articles = []articleSeed{
	{
		Title:      "The Future of Renewable Energy in 2026",
		Content:    "Renewable energy continues to transform the global landscape...",
		CategoryID: categories[2].ID,
		Tags:       []string{"energy", "climate", "technology"},
	},
	{
		Title:      "Global Markets Rally as Tech Stocks Surge",
		Content:    "Stock markets worldwide saw significant gains today...",
		CategoryID: categories[4].ID,
		Tags:       []string{"markets", "stocks", "economy"},
	},
	{
		Title:      "New Study Reveals Breakthrough in Quantum Computing",
		Content:    "Researchers have announced a major advancement in quantum computing...",
		CategoryID: categories[2].ID,
		Tags:       []string{"quantum", "computing", "research"},
	},
	{
		Title:      "Championship Finals Set New Viewership Record",
		Content:    "This year's championship finals shattered previous viewership records...",
		CategoryID: categories[3].ID,
		Tags:       []string{"sports", "championship", "records"},
	},
	{
		Title:      "Election Season Heats Up Across Major Economies",
		Content:    "As election season approaches, candidates are ramping up their campaigns...",
		CategoryID: categories[1].ID,
		Tags:       []string{"election", "politics", "campaign"},
	},
	{
		Title:      "AI Regulation Debate Intensifies in Washington",
		Content:    "Lawmakers are grappling with how to regulate artificial intelligence...",
		CategoryID: categories[1].ID,
		Tags:       []string{"AI", "regulation", "policy"},
	},
	{
		Title:      "Hollywood's Biggest Blockbusters of the Summer",
		Content:    "The summer movie season is delivering record-breaking box office numbers...",
		CategoryID: categories[3].ID,
		Tags:       []string{"movies", "entertainment", "box-office"},
	},
	{
		Title:      "Small Business Owners Optimistic About Q3 Growth",
		Content:    "A new survey reveals growing confidence among small business owners...",
		CategoryID: categories[4].ID,
		Tags:       []string{"small-business", "economy", "growth"},
	},
	{
		Title:      "Community Voices: Local Initiatives Making a Difference",
		Content:    "Across the country, community-led projects are creating positive change...",
		CategoryID: categories[0].ID,
		Tags:       []string{"community", "local", "initiatives"},
	},
	{
		Title:      "Space Exploration Reaches New Milestones",
		Content:    "Both government agencies and private companies are pushing boundaries...",
		CategoryID: categories[2].ID,
		Tags:       []string{"space", "exploration", "NASA"},
	},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Article{},
		&models.Bookmark{},
		&models.Comment{},
		&models.LatestNews{},
		&models.ArticleVideo{},
	); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	for _, c := range categories {
		cat := models.Category{
			ID:       c.ID,
			Name:     c.Name,
			Slug:     c.Slug,
			IsActive: true,
		}
		if err := db.FirstOrCreate(&cat, "id = ?", c.ID).Error; err != nil {
			log.Fatalf("seed category %s: %v", c.Name, err)
		}
		fmt.Printf("  category: %s\n", c.Name)
	}

	fixedUserID := "cm0b1b2c3d4e5f6a7b8c90001"
	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	admin := models.User{
		ID:       fixedUserID,
		Email:    "admin@thecommonsvoice.com",
		Password: string(hashed),
		Name:     "Admin",
		Role:     models.RoleAdmin,
		IsActive: true,
	}
	if err := db.FirstOrCreate(&admin, "id = ?", fixedUserID).Error; err != nil {
		log.Fatalf("seed admin: %v", err)
	}
	fmt.Println("  user: admin@thecommonsvoice.com / admin123")

	for _, a := range articles {
		slug := generateSlug(a.Title)
		now := time.Now()
		art := models.Article{
			ID:          uuid.Must(uuid.NewV7()).String(),
			Title:       a.Title,
			Slug:        slug,
			Content:     a.Content,
			CategoryID:  a.CategoryID,
			AuthorID:    fixedUserID,
			Status:      models.ArticleStatusPublished,
			PublishedAt: &now,
			Tags:        models.StringArray(a.Tags),
		}
		if err := db.Where("slug = ?", slug).FirstOrCreate(&art).Error; err != nil {
			log.Fatalf("seed article %s: %v", a.Title, err)
		}
		fmt.Printf("  article: %s\n", a.Title)
	}

	fmt.Println("\nSeed complete!")
}

func generateSlug(title string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(strings.ToLower(title), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 60 {
		slug = slug[:60]
	}
	return strings.TrimRight(slug, "-")
}
