package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct{}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

// GetSchoolSettings получает настройки школы
func (h *SettingsHandler) GetSchoolSettings(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var school models.School
	if err := database.DB.First(&school, schoolID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"school": school})
}

// UpdateSchoolSettings обновляет настройки школы
func (h *SettingsHandler) UpdateSchoolSettings(c *gin.Context) {
	role, _ := c.Get("role")
	
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can update school settings"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
		Email   string `json:"email"`
		LogoURL string `json:"logo_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var school models.School
	if err := database.DB.First(&school, schoolID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}

	// Обновляем поля
	if req.Name != "" {
		school.Name = req.Name
	}
	if req.Address != "" {
		school.Address = req.Address
	}
	if req.Phone != "" {
		school.Phone = req.Phone
	}
	if req.Email != "" {
		school.Email = req.Email
	}
	if req.LogoURL != "" {
		school.LogoURL = req.LogoURL
	}

	if err := database.DB.Save(&school).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update school"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"school": school})
}

// GetSystemInfo получает системную информацию
func (h *SettingsHandler) GetSystemInfo(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	// Подсчёт статистики
	var stats struct {
		TotalUsers        int64
		TotalClasses      int64
		TotalSubjects     int64
		TotalSchedules    int64
		TotalGrades       int64
		TotalAttendance   int64
		TotalHomework     int64
		TotalAnnouncements int64
	}

	database.DB.Model(&models.User{}).Where("school_id = ?", schoolID).Count(&stats.TotalUsers)
	database.DB.Model(&models.Class{}).Where("school_id = ?", schoolID).Count(&stats.TotalClasses)
	database.DB.Model(&models.Subject{}).Where("school_id = ?", schoolID).Count(&stats.TotalSubjects)
	
	database.DB.Table("schedules").
		Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("classes.school_id = ?", schoolID).
		Count(&stats.TotalSchedules)
	
	database.DB.Table("grades").
		Joins("JOIN users ON users.id = grades.student_id").
		Where("users.school_id = ?", schoolID).
		Count(&stats.TotalGrades)
	
	database.DB.Table("attendances").
		Joins("JOIN users ON users.id = attendances.student_id").
		Where("users.school_id = ?", schoolID).
		Count(&stats.TotalAttendance)
	
	database.DB.Table("homeworks").
		Joins("JOIN schedules ON schedules.id = homeworks.schedule_id").
		Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("classes.school_id = ?", schoolID).
		Count(&stats.TotalHomework)
	
	database.DB.Model(&models.Announcement{}).Where("school_id = ?", schoolID).Count(&stats.TotalAnnouncements)

	c.JSON(http.StatusOK, gin.H{
		"version": "7.0.0",
		"stats":   stats,
	})
}

// BackupDatabase создаёт резервную копию данных школы (JSON)
func (h *SettingsHandler) BackupDatabase(c *gin.Context) {
	role, _ := c.Get("role")
	
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create backups"})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Получаем все данные школы
	var school models.School
	database.DB.Preload("Users").First(&school, schoolID)

	var classes []models.Class
	database.DB.Where("school_id = ?", schoolID).
		Preload("Students").
		Preload("HomeroomTeacher").
		Preload("Starosta").
		Find(&classes)

	var subjects []models.Subject
	database.DB.Where("school_id = ?", schoolID).
		Preload("Teachers").
		Find(&subjects)

	var announcements []models.Announcement
	database.DB.Where("school_id = ?", schoolID).Find(&announcements)

	// Формируем backup
	backup := gin.H{
		"school":        school,
		"classes":       classes,
		"subjects":      subjects,
		"announcements": announcements,
		"backup_date":   database.DB.NowFunc(),
	}

	c.JSON(http.StatusOK, gin.H{"backup": backup})
}

// GetAuditLog получает лог действий (упрощённая версия)
func (h *SettingsHandler) GetAuditLog(c *gin.Context) {
	role, _ := c.Get("role")
	
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can view audit log"})
		return
	}

	schoolID, _ := c.Get("school_id")
	limit := c.DefaultQuery("limit", "100")

	// Получаем последние действия
	type AuditEntry struct {
		Type      string
		Count     int64
		LastDate  string
	}

	var entries []AuditEntry

	// Оценки
	var gradesCount int64
	var lastGradeDate string
	database.DB.Table("grades").
		Joins("JOIN users ON users.id = grades.student_id").
		Where("users.school_id = ?", schoolID).
		Count(&gradesCount)
	database.DB.Table("grades").
		Joins("JOIN users ON users.id = grades.student_id").
		Where("users.school_id = ?", schoolID).
		Order("grades.created_at DESC").
		Limit(1).
		Pluck("grades.created_at", &lastGradeDate)
	
	entries = append(entries, AuditEntry{
		Type:     "grades",
		Count:    gradesCount,
		LastDate: lastGradeDate,
	})

	// Посещаемость
	var attendanceCount int64
	var lastAttendanceDate string
	database.DB.Table("attendances").
		Joins("JOIN users ON users.id = attendances.student_id").
		Where("users.school_id = ?", schoolID).
		Count(&attendanceCount)
	database.DB.Table("attendances").
		Joins("JOIN users ON users.id = attendances.student_id").
		Where("users.school_id = ?", schoolID).
		Order("attendances.created_at DESC").
		Limit(1).
		Pluck("attendances.created_at", &lastAttendanceDate)
	
	entries = append(entries, AuditEntry{
		Type:     "attendance",
		Count:    attendanceCount,
		LastDate: lastAttendanceDate,
	})

	// ДЗ
	var homeworkCount int64
	var lastHomeworkDate string
	database.DB.Table("homeworks").
		Joins("JOIN schedules ON schedules.id = homeworks.schedule_id").
		Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("classes.school_id = ?", schoolID).
		Count(&homeworkCount)
	database.DB.Table("homeworks").
		Joins("JOIN schedules ON schedules.id = homeworks.schedule_id").
		Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("classes.school_id = ?", schoolID).
		Order("homeworks.created_at DESC").
		Limit(1).
		Pluck("homeworks.created_at", &lastHomeworkDate)
	
	entries = append(entries, AuditEntry{
		Type:     "homework",
		Count:    homeworkCount,
		LastDate: lastHomeworkDate,
	})

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"limit":   limit,
	})
}
