package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// UpdateUserRequest структура для обновления пользователя
type UpdateUserRequest struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	MiddleName string `json:"middle_name"`
	Email      string `json:"email" binding:"omitempty,email"`
	AvatarURL  string `json:"avatar_url"`
	Role       string `json:"role"` // Админ может менять роль
	AdminTitle string `json:"admin_title"` // Должность админа (завуч, старший учитель и т.д.)
}

// ChangePasswordRequest структура для смены пароля
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ListUsers возвращает список пользователей
func (h *UserHandler) ListUsers(c *gin.Context) {
	schoolID, _ := c.Get("school_id")
	role := c.Query("role")

	query := database.DB.Where("school_id = ?", schoolID)
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetUser получает информацию о пользователе
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var user models.User
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUser обновляет информацию о пользователе
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	// Пользователь может обновлять только себя, или админ может обновлять любого
	if uint(id) != userID.(uint) && role.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Обновляем поля
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.MiddleName = req.MiddleName
	if req.Email != "" {
		user.Email = req.Email
	}
	user.AvatarURL = req.AvatarURL
	
	// Админ может менять роль
	if role == "admin" && req.Role != "" {
		// Проверка: нельзя убрать роль админа у последнего админа
		if user.Role == "admin" && req.Role != "admin" {
			var adminCount int64
			database.DB.Model(&models.User{}).
				Where("school_id = ? AND role = ?", user.SchoolID, "admin").
				Count(&adminCount)
			
			if adminCount <= 1 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Cannot change role of the last admin"})
				return
			}
		}
		user.Role = req.Role
	}
	
	// Админ может менять admin_title (должность)
	if role == "admin" && req.AdminTitle != "" {
		user.AdminTitle = req.AdminTitle
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// DeleteUser удаляет пользователя
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var user models.User
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Проверяем, не последний ли это админ
	if user.Role == "admin" {
		var adminCount int64
		database.DB.Model(&models.User{}).
			Where("school_id = ? AND role = ?", schoolID, "admin").
			Count(&adminCount)
		
		if adminCount <= 1 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete the last admin"})
			return
		}
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ChangePassword изменяет пароль пользователя
func (h *UserHandler) ChangePassword(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID, _ := c.Get("user_id")

	// Только сам пользователь может менять свой пароль
	if uint(id) != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Проверяем старый пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid old password"})
		return
	}

	// Хешируем новый пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user.PasswordHash = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
