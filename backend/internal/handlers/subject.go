package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SubjectHandler struct{}

func NewSubjectHandler() *SubjectHandler {
	return &SubjectHandler{}
}

// CreateSubjectRequest структура для создания предмета
type CreateSubjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// AssignTeachersRequest структура для назначения учителей
type AssignTeachersRequest struct {
	TeacherIDs []uint `json:"teacher_ids" binding:"required"`
}

// CreateSubject создает новый предмет
func (h *SubjectHandler) CreateSubject(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var req CreateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subject := models.Subject{
		SchoolID:    schoolID.(uint),
		Name:        req.Name,
		Description: req.Description,
	}

	if err := database.DB.Create(&subject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subject"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subject": subject})
}

// ListSubjects возвращает список предметов
func (h *SubjectHandler) ListSubjects(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var subjects []models.Subject
	if err := database.DB.Where("school_id = ?", schoolID).
		Preload("Teachers").
		Find(&subjects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subjects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}

// GetSubject получает информацию о предмете
func (h *SubjectHandler) GetSubject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).
		Preload("Teachers").
		First(&subject).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subject": subject})
}

// UpdateSubject обновляет информацию о предмете
func (h *SubjectHandler) UpdateSubject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var req CreateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).First(&subject).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	// Обновляем поля
	subject.Name = req.Name
	subject.Description = req.Description

	if err := database.DB.Save(&subject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subject"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subject": subject})
}

// DeleteSubject удаляет предмет
func (h *SubjectHandler) DeleteSubject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).First(&subject).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	if err := database.DB.Delete(&subject).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subject"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subject deleted successfully"})
}

// AssignTeachers назначает учителей на предмет
func (h *SubjectHandler) AssignTeachers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var req AssignTeachersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).
		Preload("Teachers").First(&subject).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	// Получаем учителей
	var teachers []models.User
	if err := database.DB.Where("id IN ? AND school_id = ? AND role = ?", 
		req.TeacherIDs, schoolID, "teacher").Find(&teachers).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Teachers not found"})
		return
	}

	if len(teachers) != len(req.TeacherIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some teachers not found or not valid"})
		return
	}

	// Добавляем учителей
	if err := database.DB.Model(&subject).Association("Teachers").Append(&teachers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign teachers"})
		return
	}

	// Обновляем предмет с учителями
	database.DB.Preload("Teachers").First(&subject, subject.ID)

	c.JSON(http.StatusOK, gin.H{"subject": subject, "message": "Teachers assigned successfully"})
}

// RemoveTeacher удаляет учителя с предмета
func (h *SubjectHandler) RemoveTeacher(c *gin.Context) {
	subjectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject ID"})
		return
	}

	teacherID, err := strconv.Atoi(c.Param("teacher_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid teacher ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", subjectID, schoolID).First(&subject).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	var teacher models.User
	if err := database.DB.First(&teacher, teacherID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Teacher not found"})
		return
	}

	// Удаляем учителя с предмета
	if err := database.DB.Model(&subject).Association("Teachers").Delete(&teacher); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove teacher"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Teacher removed successfully"})
}
