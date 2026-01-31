package handlers

import (
	"classkeeper/internal/config"
	"classkeeper/internal/database"
	"classkeeper/internal/middleware"
	"classkeeper/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// RegisterRequest структура для регистрации
type RegisterRequest struct {
	SchoolID       uint   `json:"school_id"` // Опционально - берём из токена если есть
	Username       string `json:"username" binding:"required,min=3,max=50"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	Role           string `json:"role" binding:"required"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	MiddleName     string `json:"middle_name"`
	TeacherSubject string `json:"teacher_subject"` // Предмет учителя
}

// LoginRequest структура для входа
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse структура ответа
type AuthResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	User         *models.User `json:"user"`
}

// Register регистрирует нового пользователя
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Если school_id не указан, берём из токена (для авторизованных пользователей)
	if req.SchoolID == 0 {
		if schoolID, exists := c.Get("school_id"); exists {
			req.SchoolID = schoolID.(uint)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "School ID is required"})
			return
		}
	}

	// Проверяем существование школы
	var school models.School
	if err := database.DB.First(&school, req.SchoolID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "School not found"})
		return
	}

	// Проверяем уникальность username и email
	var existingUser models.User
	if err := database.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
		return
	}

	// ПРОСТОЕ СОХРАНЕНИЕ ПАРОЛЯ (без хеширования)
	// ВНИМАНИЕ: Это небезопасно! Только для учебного проекта!
	
	// Создаем пользователя
	user := models.User{
		SchoolID:       req.SchoolID,
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   req.Password, // Сохраняем пароль как есть
		Role:           req.Role,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		MiddleName:     req.MiddleName,
		TeacherSubject: req.TeacherSubject,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Генерируем токен
	token, err := h.generateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User:  &user,
	})
}

// Login авторизует пользователя
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Login: JSON bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("🔐 Login attempt: username=%s", req.Username)

	// Ищем пользователя
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		log.Printf("❌ Login: User not found: %s", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	log.Printf("✅ User found: id=%d, username=%s", user.ID, user.Username)
	log.Printf("   Password from DB: %s", user.PasswordHash)
	log.Printf("   Password from request: %s", req.Password)

	// ПРОСТАЯ ПРОВЕРКА ПАРОЛЯ (без bcrypt)
	if user.PasswordHash != req.Password {
		log.Printf("❌ Login: Password mismatch for user %s", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	log.Printf("✅ Login successful for user: %s", req.Username)

	// Генерируем токен
	token, err := h.generateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  &user,
	})
}

// Me возвращает информацию о текущем пользователе
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := database.DB.Preload("School").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// generateToken создает JWT токен
func (h *AuthHandler) generateToken(user *models.User) (string, error) {
	claims := middleware.JWTClaims{
		UserID:   user.ID,
		SchoolID: user.SchoolID,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.cfg.JWT.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWT.Secret))
}
