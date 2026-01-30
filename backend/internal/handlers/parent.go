package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ParentHandler struct{}

func NewParentHandler() *ParentHandler {
	return &ParentHandler{}
}

// LinkParentToStudent связывает родителя с учеником
func (h *ParentHandler) LinkParentToStudent(c *gin.Context) {
	role, _ := c.Get("role")
	
	// Только админы могут связывать родителей с учениками
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can link parents to students"})
		return
	}

	var req struct {
		ParentID  uint `json:"parent_id" binding:"required"`
		StudentID uint `json:"student_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Проверяем родителя
	var parent models.User
	if err := database.DB.Where("id = ? AND school_id = ? AND role = ?", 
		req.ParentID, schoolID, "parent").First(&parent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Parent not found"})
		return
	}

	// Проверяем ученика
	var student models.User
	if err := database.DB.Where("id = ? AND school_id = ? AND (role = ? OR role = ?)", 
		req.StudentID, schoolID, "student", "starosta").First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	// Проверяем, не связаны ли они уже
	var count int64
	database.DB.Table("parent_students").
		Where("parent_id = ? AND student_id = ?", req.ParentID, req.StudentID).
		Count(&count)
	
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Link already exists"})
		return
	}

	// Создаём связь
	if err := database.DB.Exec("INSERT INTO parent_students (parent_id, student_id) VALUES (?, ?)", 
		req.ParentID, req.StudentID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create link"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Parent linked to student successfully"})
}

// UnlinkParentFromStudent удаляет связь родителя с учеником
func (h *ParentHandler) UnlinkParentFromStudent(c *gin.Context) {
	role, _ := c.Get("role")
	
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can unlink parents"})
		return
	}

	parentID, err := strconv.Atoi(c.Param("parent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent ID"})
		return
	}

	studentID, err := strconv.Atoi(c.Param("student_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}

	// Удаляем связь
	if err := database.DB.Exec("DELETE FROM parent_students WHERE parent_id = ? AND student_id = ?", 
		parentID, studentID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Parent unlinked from student successfully"})
}

// GetParentChildren получает список детей родителя
func (h *ParentHandler) GetParentChildren(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	parentID := userID.(uint)
	
	// Если указан ID и пользователь админ, используем его
	if c.Param("id") != "" && role == "admin" {
		id, err := strconv.Atoi(c.Param("id"))
		if err == nil {
			parentID = uint(id)
		}
	}

	// Получаем детей
	var children []models.User
	database.DB.Table("users").
		Joins("JOIN parent_students ON parent_students.student_id = users.id").
		Where("parent_students.parent_id = ?", parentID).
		Preload("School").
		Find(&children)

	c.JSON(http.StatusOK, gin.H{"children": children})
}

// GetStudentParents получает список родителей ученика
func (h *ParentHandler) GetStudentParents(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}

	// Получаем родителей
	var parents []models.User
	database.DB.Table("users").
		Joins("JOIN parent_students ON parent_students.parent_id = users.id").
		Where("parent_students.student_id = ?", studentID).
		Find(&parents)

	c.JSON(http.StatusOK, gin.H{"parents": parents})
}

// GetChildGrades получает оценки ребёнка для родителя
func (h *ParentHandler) GetChildGrades(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	childID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid child ID"})
		return
	}

	// Проверяем право доступа
	if role == "parent" {
		// Проверяем что это действительно ребёнок родителя
		var count int64
		database.DB.Table("parent_students").
			Where("parent_id = ? AND student_id = ?", userID, childID).
			Count(&count)
		
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	// Получаем оценки
	var grades []models.Grade
	database.DB.Where("student_id = ?", childID).
		Preload("Subject").
		Preload("Teacher").
		Order("date DESC").
		Limit(100).
		Find(&grades)

	// Средний балл
	var avgGrade struct {
		Average float64
	}
	database.DB.Model(&models.Grade{}).
		Where("student_id = ?", childID).
		Select("AVG(grade) as average").
		Scan(&avgGrade)

	c.JSON(http.StatusOK, gin.H{
		"grades":        grades,
		"average_grade": avgGrade.Average,
	})
}

// GetChildAttendance получает посещаемость ребёнка для родителя
func (h *ParentHandler) GetChildAttendance(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	childID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid child ID"})
		return
	}

	// Проверяем право доступа
	if role == "parent" {
		var count int64
		database.DB.Table("parent_students").
			Where("parent_id = ? AND student_id = ?", userID, childID).
			Count(&count)
		
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	query := database.DB.Where("student_id = ?", childID).
		Preload("Schedule.Subject")

	if dateFrom != "" {
		query = query.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("date <= ?", dateTo)
	}

	var attendance []models.Attendance
	query.Order("date DESC").Limit(100).Find(&attendance)

	// Статистика
	var stats struct {
		Total   int64
		Present int64
		Absent  int64
		Late    int64
		Sick    int64
		Excused int64
	}

	statsQuery := database.DB.Model(&models.Attendance{}).Where("student_id = ?", childID)
	if dateFrom != "" {
		statsQuery = statsQuery.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		statsQuery = statsQuery.Where("date <= ?", dateTo)
	}

	statsQuery.Count(&stats.Total)
	statsQuery.Where("status = ?", "present").Count(&stats.Present)
	statsQuery.Where("status = ?", "absent").Count(&stats.Absent)
	statsQuery.Where("status = ?", "late").Count(&stats.Late)
	statsQuery.Where("status = ?", "sick").Count(&stats.Sick)
	statsQuery.Where("status = ?", "excused").Count(&stats.Excused)

	var percentage float64
	if stats.Total > 0 {
		percentage = float64(stats.Present) / float64(stats.Total) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"attendance": attendance,
		"stats":      stats,
		"percentage": percentage,
	})
}

// GetChildHomework получает домашние задания ребёнка для родителя
func (h *ParentHandler) GetChildHomework(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	childID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid child ID"})
		return
	}

	// Проверяем право доступа
	if role == "parent" {
		var count int64
		database.DB.Table("parent_students").
			Where("parent_id = ? AND student_id = ?", userID, childID).
			Count(&count)
		
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	// Получаем классы ученика
	var classIDs []uint
	database.DB.Table("class_students").
		Where("user_id = ?", childID).
		Pluck("class_id", &classIDs)

	if len(classIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"homework": []models.Homework{}})
		return
	}

	// Получаем ДЗ классов
	var homework []models.Homework
	database.DB.Joins("JOIN schedules ON schedules.id = homeworks.schedule_id").
		Where("schedules.class_id IN ?", classIDs).
		Preload("Schedule.Subject").
		Preload("Teacher").
		Order("homeworks.due_date ASC").
		Limit(50).
		Find(&homework)

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}
