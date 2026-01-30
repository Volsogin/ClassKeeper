package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AttendanceHandler struct{}

func NewAttendanceHandler() *AttendanceHandler {
	return &AttendanceHandler{}
}

// BulkAttendanceRequest структура для массовой отметки посещаемости
type BulkAttendanceRequest struct {
	Records []AttendanceRecord `json:"records" binding:"required"`
}

type AttendanceRecord struct {
	StudentID    uint   `json:"student_id" binding:"required"`
	ClassID      uint   `json:"class_id" binding:"required"`
	SubjectID    *uint  `json:"subject_id,omitempty"`
	Date         string `json:"date" binding:"required"` // YYYY-MM-DD
	LessonNumber *int   `json:"lesson_number,omitempty"`
	Status       string `json:"status" binding:"required"` // present, absent, late, excused
	Comment      string `json:"comment,omitempty"`
}

// BulkMarkAttendance массовая отметка посещаемости
func (h *AttendanceHandler) BulkMarkAttendance(c *gin.Context) {
	userID, _ := c.Get("user_id")
	schoolID, _ := c.Get("school_id")
	role, _ := c.Get("role")

	var req BulkAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем права
	if role != "admin" && role != "teacher" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins and teachers can mark attendance"})
		return
	}

	var attendances []models.Attendance

	for _, record := range req.Records {
		// Проверяем класс
		var class models.Class
		if err := database.DB.Where("id = ? AND school_id = ?", record.ClassID, schoolID).First(&class).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
			return
		}

		// Проверяем ученика
		var student models.User
		if err := database.DB.Where("id = ? AND school_id = ? AND role = ?", record.StudentID, schoolID, "student").First(&student).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Student not found"})
			return
		}

		// Парсим дату
		date, err := time.Parse("2006-01-02", record.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format (use YYYY-MM-DD)"})
			return
		}

		// Проверяем существующую запись
		var existing models.Attendance
		query := database.DB.Where("student_id = ? AND class_id = ? AND date = ?", record.StudentID, record.ClassID, date)
		if record.LessonNumber != nil {
			query = query.Where("lesson_number = ?", *record.LessonNumber)
		}
		if record.SubjectID != nil {
			query = query.Where("subject_id = ?", *record.SubjectID)
		}

		if err := query.First(&existing).Error; err == nil {
			// Обновляем существующую
			existing.Status = record.Status
			existing.Comment = record.Comment
			markedByID := userID.(uint)
			existing.MarkedBy = &markedByID
			database.DB.Save(&existing)
			attendances = append(attendances, existing)
		} else {
			// Создаём новую
			markedByID := userID.(uint)
			attendance := models.Attendance{
				StudentID:    record.StudentID,
				ClassID:      record.ClassID,
				SubjectID:    record.SubjectID,
				Date:         date,
				LessonNumber: record.LessonNumber,
				Status:       record.Status,
				Comment:      record.Comment,
				MarkedBy:     &markedByID,
			}

			if err := database.DB.Create(&attendance).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attendance record"})
				return
			}
			attendances = append(attendances, attendance)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Attendance marked successfully",
		"attendance": attendances,
	})
}

// GetAttendanceByClass получает посещаемость для класса по дате
func (h *AttendanceHandler) GetAttendanceByClass(c *gin.Context) {
	classID, _ := strconv.Atoi(c.Param("classId"))
	date := c.Param("date")
	schoolID, _ := c.Get("school_id")

	// Проверяем класс
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
		return
	}

	// Парсим дату
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format (use YYYY-MM-DD)"})
		return
	}

	// Получаем посещаемость
	var attendance []models.Attendance
	query := database.DB.Where("class_id = ? AND date = ?", classID, parsedDate)

	// Фильтр по номеру урока если указан
	if lessonNumber := c.Query("lesson_number"); lessonNumber != "" {
		query = query.Where("lesson_number = ?", lessonNumber)
	}

	if err := query.
		Preload("Student").
		Preload("Class").
		Preload("Subject").
		Find(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attendance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"attendance": attendance})
}

// GetAttendanceStats получает статистику посещаемости
func (h *AttendanceHandler) GetAttendanceStats(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required"})
		return
	}

	parsedStart, _ := time.Parse("2006-01-02", startDate)
	parsedEnd, _ := time.Parse("2006-01-02", endDate)

	// Получаем статистику
	var stats []struct {
		Status string
		Count  int
	}

	if err := database.DB.Table("attendance").
		Select("status, COUNT(*) as count").
		Joins("JOIN users ON users.id = attendance.student_id").
		Where("users.school_id = ? AND attendance.date BETWEEN ? AND ?", schoolID, parsedStart, parsedEnd).
		Group("status").
		Scan(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch statistics"})
		return
	}

	result := map[string]int{
		"present": 0,
		"absent":  0,
		"late":    0,
		"excused": 0,
	}

	for _, stat := range stats {
		result[stat.Status] = stat.Count
	}

	c.JSON(http.StatusOK, gin.H{"stats": result})
}

// MarkAttendance отмечает посещаемость одного ученика (для совместимости)
func (h *AttendanceHandler) MarkAttendance(c *gin.Context) {
	// Используем BulkMarkAttendance с одной записью
	var record AttendanceRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Оборачиваем в массив
	req := BulkAttendanceRequest{
		Records: []AttendanceRecord{record},
	}

	// Подменяем request body
	c.Set("bulk_request", req)
	h.BulkMarkAttendance(c)
}

// GetAttendance получает список посещаемости с фильтрами
func (h *AttendanceHandler) GetAttendance(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	query := database.DB.Joins("JOIN users ON users.id = attendance.student_id").
		Where("users.school_id = ?", schoolID)

	// Фильтр по классу
	if classID := c.Query("class_id"); classID != "" {
		query = query.Where("attendance.class_id = ?", classID)
	}

	// Фильтр по предмету
	if subjectID := c.Query("subject_id"); subjectID != "" {
		query = query.Where("attendance.subject_id = ?", subjectID)
	}

	// Фильтр по дате
	if date := c.Query("date"); date != "" {
		query = query.Where("attendance.date = ?", date)
	}

	// Фильтр по статусу
	if status := c.Query("status"); status != "" {
		query = query.Where("attendance.status = ?", status)
	}

	var attendance []models.Attendance
	if err := query.
		Preload("Student").
		Preload("Class").
		Preload("Subject").
		Order("attendance.date DESC").
		Limit(100).
		Find(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch attendance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"attendance": attendance})
}

// GetStudentStats получает статистику посещаемости ученика
func (h *AttendanceHandler) GetStudentStats(c *gin.Context) {
	studentID, _ := strconv.Atoi(c.Param("id"))
	schoolID, _ := c.Get("school_id")

	// Проверяем ученика
	var student models.User
	if err := database.DB.Where("id = ? AND school_id = ?", studentID, schoolID).First(&student).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Student not found"})
		return
	}

	// Получаем статистику
	var stats []struct {
		Status string
		Count  int
	}

	if err := database.DB.Table("attendance").
		Select("status, COUNT(*) as count").
		Where("student_id = ?", studentID).
		Group("status").
		Scan(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch statistics"})
		return
	}

	result := map[string]int{
		"present": 0,
		"absent":  0,
		"late":    0,
		"excused": 0,
	}

	total := 0
	for _, stat := range stats {
		result[stat.Status] = stat.Count
		total += stat.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"student": student,
		"stats":   result,
		"total":   total,
	})
}

// DeleteAttendance удаляет запись посещаемости
func (h *AttendanceHandler) DeleteAttendance(c *gin.Context) {
	attendanceID, _ := strconv.Atoi(c.Param("id"))

	var attendance models.Attendance
	if err := database.DB.First(&attendance, attendanceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attendance not found"})
		return
	}

	if err := database.DB.Delete(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attendance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Attendance deleted successfully"})
}
