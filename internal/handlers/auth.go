package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/middleware"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/services"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/validator"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

// constructor for AuthHandler
func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		DB: db, Cfg: cfg,
	}
}

type loginRequest struct {
	Email string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=30"`
}

type registerRequest struct {
	Name     string `json:"name" validate:"required,min=2"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=30"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r * http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w,http.StatusBadRequest,"Invalid request body")
		return
	}

	// replacing manual validation with our validator
	// if req.Name == "" || req.Email == "" || req.Password == "" {
	// 	response.Error(w,http.StatusBadRequest,"Missing required fields")
	// 	return
	// }

	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"message": "Validation failed",
			"errors": errs,
		})
		return
	}

	// if len(req.Password) < 6 {
	// 	response.Error(w, http.StatusBadRequest, "Password must be atleast 6 characters")
	// 	return
	// }

	var existing models.User
	if  err := h.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		response.Error(w, http.StatusBadRequest, "Email already exists")
		return
	}

	hash, err := services.HashPassword(req.Password)

	if err != nil {
		response.Error(w, http.StatusInternalServerError,"Failed to hash password")
		return
	}


	user := models.User{
		Email: req.Email,
		Password: hash,
		Name: req.Name,
		Role: models.RoleUser,
		IsActive: true,
	}
	if err :=  h.DB.Create(&user).Error; err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	tokens, err := services.GenerateTokens(user.ID, string(user.Role), user.Email, h.Cfg.JWTSecret)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Store refresh token in DB
	h.DB.Create(&models.RefreshToken{
		Jti: tokens.Jti, UserID: user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})

	response.SetAccessToken(w, tokens.AccessToken)
	response.SetRefreshToken(w, tokens.RefreshToken)

	response.JSON(w, http.StatusCreated, map[string]any{
		"message": "Registration successful",
		"user": map[string]any{
			"id": user.ID,
			"name": user.Name,
			"email": user.Email,
			"role": user.Role,
		},
		"accessToken": tokens.AccessToken,
	})
}


func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest,"Invalid request body")
	   return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.JSON(w,http.StatusBadRequest, map[string]any{
			"message": "Validation failed",
			"errors": errs,
		})
		return
	}

	var user models.User

	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		response.Error(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !services.ComparePassword(user.Password, req.Password) {
		response.Error(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !user.IsActive {
		response.Error(w, http.StatusForbidden, "Account is deactivated")
		return
	}

	tokens, err := services.GenerateTokens(user.ID, string(user.Role), user.Email, h.Cfg.JWTSecret)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	h.DB.Create(&models.RefreshToken{
		Jti: tokens.Jti, UserID: user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})

	response.SetAccessToken(w, tokens.AccessToken)
	response.SetRefreshToken(w, tokens.RefreshToken)

	response.JSON(w, http.StatusOK, map[string]any{
		"message": "Login successful",
		"user": map[string]any{
			"id": user.ID,
			"name": user.Name,
			"email": user.Email,
			"role": user.Role,
		},
		"accessToken": tokens.AccessToken,
	})
} 

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var tokenStr string

	cookie, err := r.Cookie("refresh_token")

	if err == nil && cookie.Value != "" {
		tokenStr = cookie.Value
	}

	if tokenStr == "" {
		response.Error(w, http.StatusUnauthorized, "No refresh token")
		return
	}

	claims, err := services.ValidateToken(tokenStr, h.Cfg.JWTSecret)

	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	var record models.RefreshToken

	if err := h.DB.Where("jti = ? AND \"userId\" = ?", claims.ID, claims.UserID).First(&record).Error; err != nil {
		response.Error(w, http.StatusUnauthorized, "Refresh token not found")
		return
	}

	if record.Revoked{
		response.Error(w, http.StatusUnauthorized, "Refresh token revoked")
		return
	}

	h.DB.Model(&record).Update("revoked", true)

	tokens, err := services.GenerateTokens(claims.UserID,claims.Role, claims.Email, h.Cfg.JWTSecret)

	if err != nil {
		response.Error(w, http.StatusInternalServerError,"Failed to generate tokens")
		return
	}

	h.DB.Create(&models.RefreshToken{
		Jti: tokens.Jti, UserID: claims.UserID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})

	response.SetAccessToken(w,tokens.AccessToken)
	response.SetRefreshToken(w,tokens.RefreshToken)

	response.JSON(w, http.StatusOK, map[string]any{
		"message": "Refreshed",
		"accessToken": tokens.AccessToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")

	if err == nil && cookie.Value != "" {
		if claims, err := services.ValidateToken(cookie.Value,h.Cfg.JWTSecret); err == nil {
			h.DB.Model(&models.RefreshToken{}).Where("jti = ? AND \"userId\" = ?", claims.ID, claims.UserID).Update("revoked", true)
		}
	}

	response.ClearAuthCookies(w)
	response.JSON(w, http.StatusOK, map[string]string{
		"message":"Logged out successfully",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {

	user := middleware.GetUser(r)

	if user == nil {
		response.Error(w,http.StatusUnauthorized, "Not Authenticated")
		return
	}

	var u models.User

	if err := h.DB.First(&u, "id = ?", user.UserID).Error; err != nil {
		response.Error(w,http.StatusNotFound, "User Not Found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id": u.ID, "name":u.Name,
			"email":u.Email, "role":u.Role,
		},
	})
}