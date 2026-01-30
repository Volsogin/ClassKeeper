package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type HomeworkHandler struct{}

func NewHomeworkHandler() *HomeworkHandler {
	return &HomeworkHandler{}
}

// CreateHomeworkRequest структура для создания домашнего задания
type CreateHomeworkRequest struct {
	ClassID      uint   `json:"class_id" binding:"required"`
	SubjectID    uint   `json:"subject_id" binding:"required"`
	Description  string `json:"description" binding:"required"`
	AssignedDate string `json:"assigned_date" binding:"required"` // YYYY-MM-DD
	DueDate      string `json:"due_date" binding:"required"`      // YYYY-MM-DD
}

// CreateHomework создает новое домашнее задание
func (h *HomeworkHandler) CreateHomework(c *gin.Context) {
	userID, _ := c.Get("user_id")
	schoolID, _ := c.Get("school_id")
	role, _ := c.Get("role")

	var req CreateHomeworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем класс
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", req.ClassID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
		return
	}

	// Проверяем предмет
	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", req.SubjectID, schoolID).First(&subject).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject not found"})
		return
	}

	// Проверяем права (только учителя этого предмета или классный руководитель или админ)
	if role == "teacher" {
		var teacher models.User
		database.DB.First(&teacher, userID)

		// Проверяем что учитель преподаёт этот предмет
		if teacher.TeacherSubject != subject.Name {
			// Проверяем может он классный руководитель этого класса
			if class.HomeroomTeacherID == nil || *class.HomeroomTeacherID != userID.(uint) {
				c.JSON(http.StatusForbidden, gin.H{"error": "You can only create homework for your subject or your class"})
				return
			}
		}
	}

	// Парсим даты
	assignedDate, err := time.Parse("2006-01-02", req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assigned_date format (use YYYY-MM-DD)"})
		return
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due_date format (use YYYY-MM-DD)"})
		return
	}

	homework := models.Homework{
		ClassID:      req.ClassID,
		SubjectID:    req.SubjectID,
		TeacherID:    userID.(uint),
		Description:  req.Description,
		AssignedDate: assignedDate,
		DueDate:      dueDate,
	}

	if err := database.DB.Create(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create homework"})
		return
	}

	// Загружаем связи
	database.DB.Preload("Class").Preload("Subject").Preload("Teacher").First(&homework, homework.ID)

	c.JSON(http.StatusCreated, gin.H{"homework": homework})
}

// GetAllHomework получает все домашние задания для школы
func (h *HomeworkHandler) GetAllHomework(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var homework []models.Homework
	if err := database.DB.
		Joins("JOIN classes ON classes.id = homeworks.class_id").
		Where("classes.school_id = ?", schoolID).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher").
		Order("due_date ASC").
		Find(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch homework"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}

// GetHomeworkByClass получает домашние задания для класса
func (h *HomeworkHandler) GetHomeworkByClass(c *gin.Context) {
	classID, _ := strconv.Atoi(c.Param("classId"))
	schoolID, _ := c.Get("school_id")

	// Проверяем что класс принадлежит школе
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
		return
	}

	var homework []models.Homework
	if err := database.DB.
		Where("class_id = ?", classID).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher").
		Order("due_date ASC").
		Find(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch homework"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}

// GetHomeworkForStudent получает домашние задания для ученика
func (h *HomeworkHandler) GetHomeworkForStudent(c *gin.Context) {
	studentID, _ := strconv.Atoi(c.Param("studentId"))
	schoolID, _ := c.Get("school_id")

	// Получаем класс ученика
	var user models.User
	if err := database.DB.Where("id = ? AND school_id = ?", studentID, schoolID).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Student not found"})
		return
	}

	// Находим класс где ученик учится (используем many2many связь)
	var classes []models.Class
	if err := database.DB.Model(&user).Association("Students").Find(&classes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Student is not enrolled in any class"})
		return
	}

	if len(classes) == 0 {
		c.JSON(http.StatusOK, gin.H{"homework": []models.Homework{}})
		return
	}

	// Берём первый класс (обычно ученик в одном классе)
	classID := classes[0].ID

	// Получаем домашние задания для этого класса
	var homework []models.Homework
	if err := database.DB.
		Where("class_id = ?", classID).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher").
		Order("due_date ASC").
		Find(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch homework"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}

// DeleteHomework удаляет домашнее задание
func (h *HomeworkHandler) DeleteHomework(c *gin.Context) {
	homeworkID, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var homework models.Homework
	if err := database.DB.First(&homework, homeworkID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Homework not found"})
		return
	}

	// Проверяем права (только автор или админ)
	if role != "admin" && homework.TeacherID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own homework"})
		return
	}

	if err := database.DB.Delete(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete homework"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Homework deleted successfully"})
}

// ListHomework получает список всех ДЗ (алиас для GetAllHomework)
func (h *HomeworkHandler) ListHomework(c *gin.Context) {
	h.GetAllHomework(c)
}

// GetHomework получает одно ДЗ по ID
func (h *HomeworkHandler) GetHomework(c *gin.Context) {
	homeworkID, _ := strconv.Atoi(c.Param("id"))
	schoolID, _ := c.Get("school_id")

	var homework models.Homework
	if err := database.DB.
		Joins("JOIN classes ON classes.id = homeworks.class_id").
		Where("homeworks.id = ? AND classes.school_id = ?", homeworkID, schoolID).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher").
		First(&homework).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Homework not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}

// UpdateHomework обновляет ДЗ
func (h *HomeworkHandler) UpdateHomework(c *gin.Context) {
	homeworkID, _ := strconv.Atoi(c.Param("id"))
	userID, _ := c.Get("user_id")
	schoolID, _ := c.Get("school_id")
	role, _ := c.Get("role")

	var homework models.Homework
	if err := database.DB.
		Joins("JOIN classes ON classes.id = homeworks.class_id").
		Where("homeworks.id = ? AND classes.school_id = ?", homeworkID, schoolID).
		First(&homework).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Homework not found"})
		return
	}

	// Проверяем права (только автор или админ)
	if role != "admin" && homework.TeacherID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own homework"})
		return
	}

	var req CreateHomeworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Парсим даты
	assignedDate, err := time.Parse("2006-01-02", req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assigned_date format"})
		return
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due_date format"})
		return
	}

	// Обновляем
	homework.ClassID = req.ClassID
	homework.SubjectID = req.SubjectID
	homework.Description = req.Description
	homework.AssignedDate = assignedDate
	homework.DueDate = dueDate

	if err := database.DB.Save(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update homework"})
		return
	}

	database.DB.Preload("Class").Preload("Subject").Preload("Teacher").First(&homework, homework.ID)

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}

// GetUpcomingHomework получает предстоящие ДЗ для класса
func (h *HomeworkHandler) GetUpcomingHomework(c *gin.Context) {
	classID, _ := strconv.Atoi(c.Param("id"))
	schoolID, _ := c.Get("school_id")

	// Проверяем класс
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
		return
	}

	now := time.Now()
	var homework []models.Homework
	if err := database.DB.
		Where("class_id = ? AND due_date >= ?", classID, now).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher").
		Order("due_date ASC").
		Find(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch homework"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}

// GetOverdueHomework получает просроченные ДЗ для класса
func (h *HomeworkHandler) GetOverdueHomework(c *gin.Context) {
	classID, _ := strconv.Atoi(c.Param("id"))
	schoolID, _ := c.Get("school_id")

	// Проверяем класс
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).First(&class).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Class not found"})
		return
	}

	now := time.Now()
	var homework []models.Homework
	if err := database.DB.
		Where("class_id = ? AND due_date < ?", classID, now).
		Preload("Class").
		Preload("Subject").
		Preload("Teacher").
		Order("due_date DESC").
		Find(&homework).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch homework"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"homework": homework})
}
