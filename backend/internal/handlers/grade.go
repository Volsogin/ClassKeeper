package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type GradeHandler struct{}

func NewGradeHandler() *GradeHandler {
	return &GradeHandler{}
}

// CreateGradeRequest структура для создания оценки
type CreateGradeRequest struct {
	StudentID uint   `json:"student_id" binding:"required"`
	SubjectID uint   `json:"subject_id" binding:"required"`
	Grade     int    `json:"grade" binding:"required,min=1,max=5"` // 1-5 (можно расширить)
	GradeType string `json:"grade_type"`                            // homework, test, exam, oral, final
	Date      string `json:"date" binding:"required"`               // YYYY-MM-DD
	Comment   string `json:"comment"`
}

// CreateGrade выставляет оценку
func (h *GradeHandler) CreateGrade(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	// Только учителя и админы могут ставить оценки
	if role != "teacher" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only teachers and admins can create grades"})
		return
	}

	var req CreateGradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Парсим дату
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format (use YYYY-MM-DD)"})
		return
	}

	// Проверяем ученика
	var student models.User
	if err := database.DB.Where("id = ? AND (role = ? OR role = ?)", 
		req.StudentID, "student", "starosta").First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	// Проверяем предмет
	var subject models.Subject
	if err := database.DB.First(&subject, req.SubjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	// Если учитель - проверяем что он ведёт этот предмет
	if role == "teacher" {
		var count int64
		database.DB.Table("teachers_subjects").
			Where("user_id = ? AND subject_id = ?", userID, req.SubjectID).
			Count(&count)
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't teach this subject"})
			return
		}
	}

	grade := models.Grade{
		StudentID: req.StudentID,
		SubjectID: req.SubjectID,
		TeacherID: userID.(uint),
		Grade:     req.Grade,
		GradeType: req.GradeType,
		Date:      date,
		Comment:   req.Comment,
	}

	if err := database.DB.Create(&grade).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create grade"})
		return
	}

	// Загружаем связи
	database.DB.Preload("Student").Preload("Subject").Preload("Teacher").First(&grade, grade.ID)

	c.JSON(http.StatusCreated, gin.H{"grade": grade})
}

// ListGrades возвращает список оценок с фильтрами
func (h *GradeHandler) ListGrades(c *gin.Context) {
	studentID := c.Query("student_id")
	subjectID := c.Query("subject_id")
	teacherID := c.Query("teacher_id")
	gradeType := c.Query("grade_type")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	query := database.DB.Preload("Student").Preload("Subject").Preload("Teacher")

	if studentID != "" {
		query = query.Where("student_id = ?", studentID)
	}
	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}
	if teacherID != "" {
		query = query.Where("teacher_id = ?", teacherID)
	}
	if gradeType != "" {
		query = query.Where("grade_type = ?", gradeType)
	}
	if dateFrom != "" {
		query = query.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("date <= ?", dateTo)
	}

	var grades []models.Grade
	if err := query.Order("date DESC").Find(&grades).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch grades"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"grades": grades})
}

// GetGrade получает информацию об оценке
func (h *GradeHandler) GetGrade(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade ID"})
		return
	}

	var grade models.Grade
	if err := database.DB.Preload("Student").Preload("Subject").Preload("Teacher").
		First(&grade, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"grade": grade})
}

// UpdateGrade обновляет оценку
func (h *GradeHandler) UpdateGrade(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var grade models.Grade
	if err := database.DB.First(&grade, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
		return
	}

	// Только учитель который поставил оценку или админ может её изменить
	if role != "admin" && grade.TeacherID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this grade"})
		return
	}

	var req CreateGradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Парсим дату
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	// Обновляем поля
	grade.Grade = req.Grade
	grade.GradeType = req.GradeType
	grade.Date = date
	grade.Comment = req.Comment

	if err := database.DB.Save(&grade).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update grade"})
		return
	}

	// Загружаем связи
	database.DB.Preload("Student").Preload("Subject").Preload("Teacher").First(&grade, grade.ID)

	c.JSON(http.StatusOK, gin.H{"grade": grade})
}

// DeleteGrade удаляет оценку
func (h *GradeHandler) DeleteGrade(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grade ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")

	var grade models.Grade
	if err := database.DB.First(&grade, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grade not found"})
		return
	}

	// Только учитель который поставил оценку или админ может её удалить
	if role != "admin" && grade.TeacherID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this grade"})
		return
	}

	if err := database.DB.Delete(&grade).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete grade"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Grade deleted successfully"})
}

// GetStudentAverage вычисляет средний балл ученика
func (h *GradeHandler) GetStudentAverage(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}

	subjectID := c.Query("subject_id")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	query := database.DB.Model(&models.Grade{}).Where("student_id = ?", studentID)

	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}
	if dateFrom != "" {
		query = query.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("date <= ?", dateTo)
	}

	// Вычисляем средний балл
	var result struct {
		Average float64
		Count   int64
	}

	query.Select("AVG(grade) as average, COUNT(*) as count").Scan(&result)

	// Получаем средний балл по предметам
	var subjectAverages []struct {
		SubjectID   uint    `json:"subject_id"`
		SubjectName string  `json:"subject_name"`
		Average     float64 `json:"average"`
		Count       int64   `json:"count"`
	}

	database.DB.Model(&models.Grade{}).
		Select("grades.subject_id, subjects.name as subject_name, AVG(grades.grade) as average, COUNT(*) as count").
		Joins("JOIN subjects ON subjects.id = grades.subject_id").
		Where("grades.student_id = ?", studentID).
		Group("grades.subject_id, subjects.name").
		Scan(&subjectAverages)

	c.JSON(http.StatusOK, gin.H{
		"student_id":        studentID,
		"overall_average":   result.Average,
		"total_grades":      result.Count,
		"subject_averages": subjectAverages,
	})
}

// GetClassJournal получает журнал успеваемости класса
func (h *GradeHandler) GetClassJournal(c *gin.Context) {
	classID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	subjectID := c.Query("subject_id")

	// Получаем класс с учениками
	var class models.Class
	if err := database.DB.Preload("Students").First(&class, classID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	// Собираем ID учеников
	studentIDs := make([]uint, len(class.Students))
	for i, s := range class.Students {
		studentIDs[i] = s.ID
	}

	// Получаем оценки
	query := database.DB.Where("student_id IN ?", studentIDs).
		Preload("Student").
		Preload("Subject")

	if subjectID != "" {
		query = query.Where("subject_id = ?", subjectID)
	}

	var grades []models.Grade
	query.Order("date DESC").Find(&grades)

	// Группируем по ученикам
	journal := make(map[uint][]models.Grade)
	for _, grade := range grades {
		journal[grade.StudentID] = append(journal[grade.StudentID], grade)
	}

	c.JSON(http.StatusOK, gin.H{
		"class":   class,
		"journal": journal,
	})
}
