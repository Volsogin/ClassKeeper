package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct{}

func NewScheduleHandler() *ScheduleHandler {
	return &ScheduleHandler{}
}

// CreateScheduleRequest структура для создания урока
type CreateScheduleRequest struct {
	ClassID      uint   `json:"class_id" binding:"required"`
	SubjectID    uint   `json:"subject_id" binding:"required"`
	DayOfWeek    string `json:"day_of_week" binding:"required"` // Понедельник, Вторник...
	LessonNumber int    `json:"lesson_number" binding:"required,min=1,max=10"`
	StartTime    string `json:"start_time" binding:"required"` // HH:MM
	EndTime      string `json:"end_time" binding:"required"`   // HH:MM
	RoomNumber   string `json:"room_number,omitempty"`
}

// CreateSchedule создает новый урок в расписании
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем что класс принадлежит школе
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", req.ClassID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
		return
	}

	// Проверяем что предмет принадлежит школе
	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", req.SubjectID, schoolID).First(&subject).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject not found"})
		return
	}

	// Парсим время - просто используем строки как есть
	schedule := models.Schedule{
		ClassID:      req.ClassID,
		SubjectID:    req.SubjectID,
		DayOfWeek:    req.DayOfWeek,
		LessonNumber: req.LessonNumber,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		RoomNumber:   req.RoomNumber,
	}

	if err := database.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule"})
		return
	}

	// Загружаем связи
	database.DB.Preload("Class").Preload("Subject").First(&schedule, schedule.ID)

	c.JSON(http.StatusCreated, gin.H{"schedule": schedule})
}

// ListSchedules возвращает список уроков
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	schoolID, _ := c.Get("school_id")
	classID := c.Query("class_id")
	dayOfWeek := c.Query("day_of_week")

	query := database.DB.Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("classes.school_id = ?", schoolID).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher")

	if classID != "" {
		query = query.Where("schedules.class_id = ?", classID)
	}

	if dayOfWeek != "" {
		query = query.Where("schedules.day_of_week = ?", dayOfWeek)
	}

	var schedules []models.Schedule
	if err := query.Order("day_of_week, lesson_number").Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// GetSchedule получает информацию об уроке
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var schedule models.Schedule
	if err := database.DB.Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("schedules.id = ? AND classes.school_id = ?", id, schoolID).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher").
		First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedule": schedule})
}

// UpdateSchedule обновляет информацию об уроке
// DeleteSchedule удаляет урок
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var schedule models.Schedule
	if err := database.DB.Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("schedules.id = ? AND classes.school_id = ?", id, schoolID).
		First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	if err := database.DB.Delete(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}

// GetClassSchedule получает расписание для класса на неделю
func (h *ScheduleHandler) GetClassSchedule(c *gin.Context) {
	classID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Проверяем что класс принадлежит школе
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	// Получаем расписание
	var schedules []models.Schedule
	if err := database.DB.Where("class_id = ?", classID).
		Preload("Subject").
		Order("lesson_number").
		Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule"})
		return
	}

	// Группируем по дням недели
	weekSchedule := make(map[string][]models.Schedule)
	for _, s := range schedules {
		weekSchedule[s.DayOfWeek] = append(weekSchedule[s.DayOfWeek], s)
	}

	c.JSON(http.StatusOK, gin.H{
		"class":    class,
		"schedule": weekSchedule,
	})
}

// UpdateSchedule обновляет расписание
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var schedule models.Schedule
	if err := database.DB.Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("schedules.id = ? AND classes.school_id = ?", id, schoolID).
		First(&schedule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	// Обновляем - работаем со строками напрямую
	schedule.ClassID = req.ClassID
	schedule.SubjectID = req.SubjectID
	schedule.DayOfWeek = req.DayOfWeek
	schedule.LessonNumber = req.LessonNumber
	schedule.StartTime = req.StartTime
	schedule.EndTime = req.EndTime
	schedule.RoomNumber = req.RoomNumber

	if err := database.DB.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
		return
	}

	database.DB.Preload("Class").Preload("Subject").Preload("Teacher").First(&schedule, schedule.ID)

	c.JSON(http.StatusOK, gin.H{"schedule": schedule})
}
