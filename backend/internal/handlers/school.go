package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SchoolHandler struct{}

func NewSchoolHandler() *SchoolHandler {
	return &SchoolHandler{}
}

// CreateSchoolRequest структура для создания школы
type CreateSchoolRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email" binding:"omitempty,email"`
	LogoURL string `json:"logo_url"`
}

// CreateSchool создает новую школу
func (h *SchoolHandler) CreateSchool(c *gin.Context) {
	var req CreateSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	school := models.School{
		Name:    req.Name,
		Address: req.Address,
		Phone:   req.Phone,
		Email:   req.Email,
		LogoURL: req.LogoURL,
	}

	if err := database.DB.Create(&school).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create school"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"school": school})
}

// GetSchool получает информацию о школе
func (h *SchoolHandler) GetSchool(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid school ID"})
		return
	}

	var school models.School
	if err := database.DB.First(&school, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"school": school})
}

// UpdateSchool обновляет информацию о школе
func (h *SchoolHandler) UpdateSchool(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid school ID"})
		return
	}

	// Проверяем права доступа (только админ школы)
	schoolID, _ := c.Get("school_id")
	if uint(id) != schoolID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req CreateSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var school models.School
	if err := database.DB.First(&school, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}

	// Обновляем поля
	school.Name = req.Name
	school.Address = req.Address
	school.Phone = req.Phone
	school.Email = req.Email
	school.LogoURL = req.LogoURL

	if err := database.DB.Save(&school).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update school"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"school": school})
}

// ListSchools возвращает список всех школ (для супер-админа)
func (h *SchoolHandler) ListSchools(c *gin.Context) {
	var schools []models.School
	if err := database.DB.Find(&schools).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schools"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schools": schools})
}
