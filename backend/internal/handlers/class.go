package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ClassHandler struct{}

func NewClassHandler() *ClassHandler {
	return &ClassHandler{}
}

// CreateClassRequest структура для создания класса
type CreateClassRequest struct {
	Name              string `json:"name" binding:"required"`               // "9А", "11Б"
	Year              string `json:"year" binding:"required"`               // "2025-2026"
	HomeroomTeacherID *uint  `json:"homeroom_teacher_id,omitempty"`
	StarostaID        *uint  `json:"starosta_id,omitempty"`
}

// UpdateClassRequest структура для обновления класса
type UpdateClassRequest struct {
	Name              string `json:"name"`
	Year              string `json:"year"`
	HomeroomTeacherID *uint  `json:"homeroom_teacher_id"`
	StarostaID        *uint  `json:"starosta_id"`
}

// AddStudentsRequest структура для добавления учеников
type AddStudentsRequest struct {
	StudentIDs []uint `json:"student_ids" binding:"required"`
}

// CreateClass создает новый класс
func (h *ClassHandler) CreateClass(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var req CreateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем классного руководителя если указан
	if req.HomeroomTeacherID != nil {
		var teacher models.User
		if err := database.DB.Where("id = ? AND school_id = ? AND role = ?", 
			req.HomeroomTeacherID, schoolID, "teacher").First(&teacher).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Homeroom teacher not found or not a teacher"})
			return
		}
	}

	// Проверяем старосту если указан
	if req.StarostaID != nil {
		var starosta models.User
		if err := database.DB.Where("id = ? AND school_id = ? AND (role = ? OR role = ?)", 
			req.StarostaID, schoolID, "student", "starosta").First(&starosta).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Starosta not found or not a student"})
			return
		}
	}

	class := models.Class{
		SchoolID:          schoolID.(uint),
		Name:              req.Name,
		Year:              req.Year,
		HomeroomTeacherID: req.HomeroomTeacherID,
		StarostaID:        req.StarostaID,
	}

	if err := database.DB.Create(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create class"})
		return
	}

	// Загружаем связи
	database.DB.Preload("HomeroomTeacher").Preload("Starosta").First(&class, class.ID)

	c.JSON(http.StatusCreated, gin.H{"class": class})
}

// ListClasses возвращает список классов
func (h *ClassHandler) ListClasses(c *gin.Context) {
	schoolID, _ := c.Get("school_id")
	year := c.Query("year")

	query := database.DB.Where("school_id = ?", schoolID).
		Preload("HomeroomTeacher").
		Preload("Starosta")

	if year != "" {
		query = query.Where("year = ?", year)
	}

	var classes []models.Class
	if err := query.Find(&classes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch classes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"classes": classes})
}

// GetClass получает информацию о классе
func (h *ClassHandler) GetClass(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).
		Preload("HomeroomTeacher").
		Preload("Starosta").
		Preload("Students").
		First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"class": class})
}

// UpdateClass обновляет информацию о классе
func (h *ClassHandler) UpdateClass(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	var req UpdateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Обновляем поля
	if req.Name != "" {
		class.Name = req.Name
	}
	if req.Year != "" {
		class.Year = req.Year
	}
	if req.HomeroomTeacherID != nil {
		// Проверяем что это учитель
		var teacher models.User
		if err := database.DB.Where("id = ? AND school_id = ? AND role = ?",
			req.HomeroomTeacherID, schoolID, "teacher").First(&teacher).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid teacher"})
			return
		}
		class.HomeroomTeacherID = req.HomeroomTeacherID
	}
	if req.StarostaID != nil {
		// Проверяем что это ученик или староста
		var student models.User
		if err := database.DB.Where("id = ? AND school_id = ? AND (role = ? OR role = ?)",
			req.StarostaID, schoolID, "student", "starosta").First(&student).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student"})
			return
		}
		class.StarostaID = req.StarostaID
	}

	if err := database.DB.Save(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update class"})
		return
	}

	// Загружаем обновлённые данные
	database.DB.Preload("HomeroomTeacher").Preload("Starosta").Preload("Students").First(&class, class.ID)

	c.JSON(http.StatusOK, gin.H{"class": class})
}

// DeleteClass удаляет класс
func (h *ClassHandler) DeleteClass(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	if err := database.DB.Delete(&class).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete class"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Class deleted successfully"})
}

// AddStudents добавляет учеников в класс
func (h *ClassHandler) AddStudents(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var req AddStudentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", id, schoolID).
		Preload("Students").First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	// Получаем учеников
	var students []models.User
	if err := database.DB.Where("id IN ? AND school_id = ? AND (role = ? OR role = ?)", 
		req.StudentIDs, schoolID, "student", "starosta").Find(&students).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Students not found"})
		return
	}

	if len(students) != len(req.StudentIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some students not found or not valid"})
		return
	}

	// Добавляем учеников
	if err := database.DB.Model(&class).Association("Students").Append(&students); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add students"})
		return
	}

	// Обновляем класс с учениками
	database.DB.Preload("Students").First(&class, class.ID)

	c.JSON(http.StatusOK, gin.H{"class": class, "message": "Students added successfully"})
}

// RemoveStudent удаляет ученика из класса
func (h *ClassHandler) RemoveStudent(c *gin.Context) {
	classID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	studentID, err := strconv.Atoi(c.Param("student_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	var student models.User
	if err := database.DB.First(&student, studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	// Удаляем ученика из класса
	if err := database.DB.Model(&class).Association("Students").Delete(&student); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove student"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Student removed successfully"})
}
