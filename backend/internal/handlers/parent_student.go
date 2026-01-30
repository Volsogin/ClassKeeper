package handlers

import (
	"net/http"
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"github.com/gin-gonic/gin"
)

type ParentStudentHandler struct{}

func NewParentStudentHandler() *ParentStudentHandler {
	return &ParentStudentHandler{}
}

// CreateLink создаёт связь родитель-ученик
func (h *ParentStudentHandler) CreateLink(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var req struct {
		ParentID  uint `json:"parent_id" binding:"required"`
		StudentID uint `json:"student_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем что родитель существует и имеет роль parent
	var parent models.User
	if err := database.DB.Where("id = ? AND school_id = ? AND role = ?", 
		req.ParentID, schoolID, "parent").First(&parent).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parent not found"})
		return
	}

	// Проверяем что ученик существует и имеет роль student
	var student models.User
	if err := database.DB.Where("id = ? AND school_id = ? AND role = ?", 
		req.StudentID, schoolID, "student").First(&student).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Student not found"})
		return
	}

	// Проверяем что связь ещё не существует
	var existingLink models.ParentStudent
	if err := database.DB.Where("parent_id = ? AND student_id = ?", 
		req.ParentID, req.StudentID).First(&existingLink).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Link already exists"})
		return
	}

	// Создаём связь
	link := models.ParentStudent{
		ParentID:  req.ParentID,
		StudentID: req.StudentID,
	}

	if err := database.DB.Create(&link).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"link": link})
}

// ListLinks возвращает все связи для школы
func (h *ParentStudentHandler) ListLinks(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var links []models.ParentStudent
	
	// Preload parent и student с их данными
	if err := database.DB.Preload("Parent").Preload("Student").
		Joins("JOIN users as parents ON parents.id = parent_students.parent_id").
		Joins("JOIN users as students ON students.id = parent_students.student_id").
		Where("parents.school_id = ? AND students.school_id = ?", schoolID, schoolID).
		Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch links"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"links": links})
}

// DeleteLink удаляет связь
func (h *ParentStudentHandler) DeleteLink(c *gin.Context) {
	schoolID, _ := c.Get("school_id")
	linkID := c.Param("id")

	var link models.ParentStudent
	
	// Проверяем что связь существует и принадлежит этой школе
	if err := database.DB.Preload("Parent").Preload("Student").First(&link, linkID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
		return
	}

	// Проверяем school_id родителя
	if link.Parent.SchoolID != schoolID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := database.DB.Delete(&link).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Link deleted"})
}

// GetStudentsByParent возвращает детей родителя
func (h *ParentStudentHandler) GetStudentsByParent(c *gin.Context) {
	parentID := c.Param("parent_id")

	var links []models.ParentStudent
	if err := database.DB.Preload("Student").Where("parent_id = ?", parentID).Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch students"})
		return
	}

	var students []models.User
	for _, link := range links {
		students = append(students, link.Student)
	}

	c.JSON(http.StatusOK, gin.H{"students": students})
}

// GetParentsByStudent возвращает родителей ученика
func (h *ParentStudentHandler) GetParentsByStudent(c *gin.Context) {
	studentID := c.Param("student_id")

	var links []models.ParentStudent
	if err := database.DB.Preload("Parent").Where("student_id = ?", studentID).Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch parents"})
		return
	}

	var parents []models.User
	for _, link := range links {
		parents = append(parents, link.Parent)
	}

	c.JSON(http.StatusOK, gin.H{"parents": parents})
}
