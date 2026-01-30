package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AnnouncementHandler struct{}

func NewAnnouncementHandler() *AnnouncementHandler {
	return &AnnouncementHandler{}
}

// CreateAnnouncementRequest структура для создания объявления
type CreateAnnouncementRequest struct {
	Title         string `json:"title" binding:"required"`
	Content       string `json:"content" binding:"required"`
	TargetRole    string `json:"target_role"`    // all, teachers, students, parents
	TargetClassID *uint  `json:"target_class_id,omitempty"`
}

// CreateAnnouncement создает объявление
func (h *AnnouncementHandler) CreateAnnouncement(c *gin.Context) {
	userID, _ := c.Get("user_id")
	schoolID, _ := c.Get("school_id")
	role, _ := c.Get("role")

	// Только админы и учителя могут создавать объявления
	if role != "admin" && role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins and teachers can create announcements"})
		return
	}

	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация target_role
	validRoles := map[string]bool{
		"all": true, "teachers": true, "students": true, "parents": true,
	}
	if req.TargetRole != "" && !validRoles[req.TargetRole] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target role"})
		return
	}

	// Проверяем класс если указан
	if req.TargetClassID != nil {
		var class models.Class
		if err := database.DB.Where("id = ? AND school_id = ?", req.TargetClassID, schoolID).
			First(&class).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target class not found"})
			return
		}
	}

	announcement := models.Announcement{
		SchoolID:      schoolID.(uint),
		AuthorID:      userID.(uint),
		Title:         req.Title,
		Content:       req.Content,
		TargetRole:    req.TargetRole,
		TargetClassID: req.TargetClassID,
	}

	if err := database.DB.Create(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create announcement"})
		return
	}

	// Загружаем связи
	database.DB.Preload("Author").Preload("TargetClass").First(&announcement, announcement.ID)

	c.JSON(http.StatusCreated, gin.H{"announcement": announcement})
}

// ListAnnouncements возвращает список объявлений
func (h *AnnouncementHandler) ListAnnouncements(c *gin.Context) {
	schoolID, _ := c.Get("school_id")
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("role")

	targetRole := c.Query("target_role")
	classID := c.Query("class_id")
	limit := c.DefaultQuery("limit", "50")

	query := database.DB.Where("school_id = ?", schoolID).
		Preload("Author").
		Preload("TargetClass")

	// Фильтрация по правам доступа
	// Пользователь видит объявления для:
	// 1. Всех (target_role = "all")
	// 2. Своей роли (target_role = userRole)
	// 3. Своего класса (если ученик/родитель)
	
	if userRole != "admin" {
		// Получаем классы пользователя если он ученик
		var userClasses []uint
		if userRole == "student" || userRole == "starosta" {
			var classes []models.Class
			database.DB.Joins("JOIN class_students ON class_students.class_id = classes.id").
				Where("class_students.user_id = ?", userID).
				Find(&classes)
			for _, class := range classes {
				userClasses = append(userClasses, class.ID)
			}
		}

		// Строим условие фильтрации
		if len(userClasses) > 0 {
			query = query.Where(
				"target_role = ? OR target_role = ? OR target_class_id IN ?",
				"all", userRole, userClasses,
			)
		} else {
			query = query.Where("target_role = ? OR target_role = ?", "all", userRole)
		}
	}

	// Дополнительные фильтры
	if targetRole != "" {
		query = query.Where("target_role = ?", targetRole)
	}
	if classID != "" {
		query = query.Where("target_class_id = ?", classID)
	}

	var announcements []models.Announcement
	if err := query.Order("created_at DESC").Limit(parseInt(limit)).
		Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch announcements"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"announcements": announcements})
}

// GetAnnouncement получает информацию об объявлении
func (h *AnnouncementHandler) GetAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid announcement ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var announcement models.Announcement
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).
		Preload("Author").
		Preload("TargetClass").
		First(&announcement).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Announcement not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"announcement": announcement})
}

// UpdateAnnouncement обновляет объявление
func (h *AnnouncementHandler) UpdateAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid announcement ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var announcement models.Announcement
	if err := database.DB.First(&announcement, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Announcement not found"})
		return
	}

	// Только автор или админ может редактировать
	if role != "admin" && announcement.AuthorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this announcement"})
		return
	}

	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Обновляем поля
	announcement.Title = req.Title
	announcement.Content = req.Content
	announcement.TargetRole = req.TargetRole
	announcement.TargetClassID = req.TargetClassID

	if err := database.DB.Save(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update announcement"})
		return
	}

	// Загружаем связи
	database.DB.Preload("Author").Preload("TargetClass").First(&announcement, announcement.ID)

	c.JSON(http.StatusOK, gin.H{"announcement": announcement})
}

// DeleteAnnouncement удаляет объявление
func (h *AnnouncementHandler) DeleteAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid announcement ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var announcement models.Announcement
	if err := database.DB.First(&announcement, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Announcement not found"})
		return
	}

	// Только автор или админ может удалять
	if role != "admin" && announcement.AuthorID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this announcement"})
		return
	}

	if err := database.DB.Delete(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete announcement"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Announcement deleted successfully"})
}

// GetMyAnnouncements получает объявления созданные текущим пользователем
func (h *AnnouncementHandler) GetMyAnnouncements(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var announcements []models.Announcement
	if err := database.DB.Where("author_id = ?", userID).
		Preload("TargetClass").
		Order("created_at DESC").
		Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch announcements"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"announcements": announcements})
}

// GetClassAnnouncements получает объявления для конкретного класса
func (h *AnnouncementHandler) GetClassAnnouncements(c *gin.Context) {
	classID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Проверяем класс
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).
		First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	// Получаем объявления для класса и для всех
	var announcements []models.Announcement
	if err := database.DB.Where("school_id = ? AND (target_class_id = ? OR target_role = ?)", 
		schoolID, classID, "all").
		Preload("Author").
		Order("created_at DESC").
		Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch announcements"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"class":         class,
		"announcements": announcements,
	})
}

// Helper функция для парсинга int
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	if i <= 0 {
		return 50
	}
	return i
}
