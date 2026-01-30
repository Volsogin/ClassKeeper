package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct{}

func NewAnalyticsHandler() *AnalyticsHandler {
	return &AnalyticsHandler{}
}

// GetSchoolStats получает общую статистику школы
func (h *AnalyticsHandler) GetSchoolStats(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	var stats struct {
		TotalClasses   int64
		TotalStudents  int64
		TotalTeachers  int64
		TotalSubjects  int64
		TotalSchedules int64
		TotalGrades    int64
		TotalHomework  int64
	}

	// Считаем классы
	database.DB.Model(&models.Class{}).Where("school_id = ?", schoolID).Count(&stats.TotalClasses)

	// Считаем учеников
	database.DB.Model(&models.User{}).Where("school_id = ? AND (role = ? OR role = ?)", 
		schoolID, "student", "starosta").Count(&stats.TotalStudents)

	// Считаем учителей
	database.DB.Model(&models.User{}).Where("school_id = ? AND role = ?", 
		schoolID, "teacher").Count(&stats.TotalTeachers)

	// Считаем предметы
	database.DB.Model(&models.Subject{}).Where("school_id = ?", schoolID).Count(&stats.TotalSubjects)

	// Считаем расписание
	database.DB.Table("schedules").
		Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("classes.school_id = ?", schoolID).
		Count(&stats.TotalSchedules)

	// Считаем оценки
	database.DB.Table("grades").
		Joins("JOIN users ON users.id = grades.student_id").
		Where("users.school_id = ?", schoolID).
		Count(&stats.TotalGrades)

	// Считаем ДЗ
	database.DB.Table("homeworks").
		Joins("JOIN schedules ON schedules.id = homeworks.schedule_id").
		Joins("JOIN classes ON classes.id = schedules.class_id").
		Where("classes.school_id = ?", schoolID).
		Count(&stats.TotalHomework)

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// GetClassStats получает статистику класса
func (h *AnalyticsHandler) GetClassStats(c *gin.Context) {
	classID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Проверяем класс
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).
		Preload("Students").First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	// Средний балл класса
	var avgGrade struct {
		Average float64
	}
	database.DB.Table("grades").
		Joins("JOIN users ON users.id = grades.student_id").
		Joins("JOIN class_students ON class_students.user_id = users.id").
		Where("class_students.class_id = ?", classID).
		Select("AVG(grades.grade) as average").
		Scan(&avgGrade)

	// Посещаемость класса
	var attendance struct {
		Total   int64
		Present int64
	}
	database.DB.Table("attendances").
		Joins("JOIN schedules ON schedules.id = attendances.schedule_id").
		Where("schedules.class_id = ?", classID).
		Count(&attendance.Total)

	database.DB.Table("attendances").
		Joins("JOIN schedules ON schedules.id = attendances.schedule_id").
		Where("schedules.class_id = ? AND attendances.status = ?", classID, "present").
		Count(&attendance.Present)

	var attendancePercentage float64
	if attendance.Total > 0 {
		attendancePercentage = float64(attendance.Present) / float64(attendance.Total) * 100
	}

	// Количество уроков в неделю
	var lessonsPerWeek int64
	database.DB.Model(&models.Schedule{}).Where("class_id = ?", classID).Count(&lessonsPerWeek)

	c.JSON(http.StatusOK, gin.H{
		"class":                 class,
		"total_students":        len(class.Students),
		"average_grade":         avgGrade.Average,
		"attendance_percentage": attendancePercentage,
		"lessons_per_week":      lessonsPerWeek,
	})
}

// GetAttendanceReport получает детальный отчёт по посещаемости
func (h *AnalyticsHandler) GetAttendanceReport(c *gin.Context) {
	schoolID, _ := c.Get("school_id")
	classID := c.Query("class_id")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02") // Последний месяц
	}
	if dateTo == "" {
		dateTo = time.Now().Format("2006-01-02")
	}

	type AttendanceReport struct {
		StudentID   uint    `json:"student_id"`
		StudentName string  `json:"student_name"`
		ClassName   string  `json:"class_name"`
		Total       int64   `json:"total"`
		Present     int64   `json:"present"`
		Absent      int64   `json:"absent"`
		Late        int64   `json:"late"`
		Sick        int64   `json:"sick"`
		Excused     int64   `json:"excused"`
		Percentage  float64 `json:"percentage"`
	}

	query := `
		SELECT 
			users.id as student_id,
			CONCAT(users.first_name, ' ', users.last_name) as student_name,
			classes.name as class_name,
			COUNT(*) as total,
			SUM(CASE WHEN attendances.status = 'present' THEN 1 ELSE 0 END) as present,
			SUM(CASE WHEN attendances.status = 'absent' THEN 1 ELSE 0 END) as absent,
			SUM(CASE WHEN attendances.status = 'late' THEN 1 ELSE 0 END) as late,
			SUM(CASE WHEN attendances.status = 'sick' THEN 1 ELSE 0 END) as sick,
			SUM(CASE WHEN attendances.status = 'excused' THEN 1 ELSE 0 END) as excused,
			(SUM(CASE WHEN attendances.status = 'present' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as percentage
		FROM attendances
		JOIN users ON users.id = attendances.student_id
		JOIN schedules ON schedules.id = attendances.schedule_id
		JOIN classes ON classes.id = schedules.class_id
		WHERE users.school_id = ?
			AND attendances.date BETWEEN ? AND ?
	`

	args := []interface{}{schoolID, dateFrom, dateTo}

	if classID != "" {
		query += " AND classes.id = ?"
		args = append(args, classID)
	}

	query += " GROUP BY users.id, student_name, class_name ORDER BY percentage DESC"

	var reports []AttendanceReport
	database.DB.Raw(query, args...).Scan(&reports)

	c.JSON(http.StatusOK, gin.H{
		"date_from": dateFrom,
		"date_to":   dateTo,
		"report":    reports,
	})
}

// GetGradesReport получает детальный отчёт по оценкам
func (h *AnalyticsHandler) GetGradesReport(c *gin.Context) {
	schoolID, _ := c.Get("school_id")
	classID := c.Query("class_id")
	subjectID := c.Query("subject_id")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = time.Now().Format("2006-01-02")
	}

	type GradesReport struct {
		StudentID   uint    `json:"student_id"`
		StudentName string  `json:"student_name"`
		ClassName   string  `json:"class_name"`
		SubjectName string  `json:"subject_name"`
		Average     float64 `json:"average"`
		Count       int64   `json:"count"`
		Grade5      int64   `json:"grade_5"`
		Grade4      int64   `json:"grade_4"`
		Grade3      int64   `json:"grade_3"`
		Grade2      int64   `json:"grade_2"`
	}

	query := `
		SELECT 
			users.id as student_id,
			CONCAT(users.first_name, ' ', users.last_name) as student_name,
			classes.name as class_name,
			subjects.name as subject_name,
			AVG(grades.grade) as average,
			COUNT(*) as count,
			SUM(CASE WHEN grades.grade = 5 THEN 1 ELSE 0 END) as grade_5,
			SUM(CASE WHEN grades.grade = 4 THEN 1 ELSE 0 END) as grade_4,
			SUM(CASE WHEN grades.grade = 3 THEN 1 ELSE 0 END) as grade_3,
			SUM(CASE WHEN grades.grade = 2 THEN 1 ELSE 0 END) as grade_2
		FROM grades
		JOIN users ON users.id = grades.student_id
		JOIN subjects ON subjects.id = grades.subject_id
		JOIN class_students ON class_students.user_id = users.id
		JOIN classes ON classes.id = class_students.class_id
		WHERE users.school_id = ?
			AND grades.date BETWEEN ? AND ?
	`

	args := []interface{}{schoolID, dateFrom, dateTo}

	if classID != "" {
		query += " AND classes.id = ?"
		args = append(args, classID)
	}
	if subjectID != "" {
		query += " AND subjects.id = ?"
		args = append(args, subjectID)
	}

	query += " GROUP BY users.id, student_name, class_name, subject_name ORDER BY average DESC"

	var reports []GradesReport
	database.DB.Raw(query, args...).Scan(&reports)

	c.JSON(http.StatusOK, gin.H{
		"date_from": dateFrom,
		"date_to":   dateTo,
		"report":    reports,
	})
}

// GetTeacherStats получает статистику учителя
func (h *AnalyticsHandler) GetTeacherStats(c *gin.Context) {
	teacherID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid teacher ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Проверяем учителя
	var teacher models.User
	if err := database.DB.Where("id = ? AND school_id = ? AND role = ?", 
		teacherID, schoolID, "teacher").First(&teacher).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Teacher not found"})
		return
	}

	// Количество уроков
	var lessonsCount int64
	database.DB.Model(&models.Schedule{}).Where("teacher_id = ?", teacherID).Count(&lessonsCount)

	// Количество классов
	var classesCount int64
	database.DB.Table("schedules").
		Where("teacher_id = ?", teacherID).
		Distinct("class_id").
		Count(&classesCount)

	// Предметы которые ведёт
	var subjects []models.Subject
	database.DB.Table("subjects").
		Joins("JOIN teachers_subjects ON teachers_subjects.subject_id = subjects.id").
		Where("teachers_subjects.user_id = ?", teacherID).
		Find(&subjects)

	// Средний балл по оценкам
	var avgGrade struct {
		Average float64
		Count   int64
	}
	database.DB.Model(&models.Grade{}).
		Where("teacher_id = ?", teacherID).
		Select("AVG(grade) as average, COUNT(*) as count").
		Scan(&avgGrade)

	// Количество ДЗ
	var homeworkCount int64
	database.DB.Model(&models.Homework{}).Where("teacher_id = ?", teacherID).Count(&homeworkCount)

	c.JSON(http.StatusOK, gin.H{
		"teacher":         teacher,
		"lessons_count":   lessonsCount,
		"classes_count":   classesCount,
		"subjects":        subjects,
		"average_grade":   avgGrade.Average,
		"total_grades":    avgGrade.Count,
		"homework_count":  homeworkCount,
	})
}

// GetSubjectStats получает статистику по предмету
func (h *AnalyticsHandler) GetSubjectStats(c *gin.Context) {
	subjectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Проверяем предмет
	var subject models.Subject
	if err := database.DB.Where("id = ? AND school_id = ?", subjectID, schoolID).
		Preload("Teachers").First(&subject).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subject not found"})
		return
	}

	// Средний балл
	var avgGrade struct {
		Average float64
		Count   int64
	}
	database.DB.Model(&models.Grade{}).
		Where("subject_id = ?", subjectID).
		Select("AVG(grade) as average, COUNT(*) as count").
		Scan(&avgGrade)

	// Распределение оценок
	var gradeDistribution struct {
		Grade5 int64
		Grade4 int64
		Grade3 int64
		Grade2 int64
		Grade1 int64
	}
	database.DB.Model(&models.Grade{}).Where("subject_id = ? AND grade = ?", subjectID, 5).Count(&gradeDistribution.Grade5)
	database.DB.Model(&models.Grade{}).Where("subject_id = ? AND grade = ?", subjectID, 4).Count(&gradeDistribution.Grade4)
	database.DB.Model(&models.Grade{}).Where("subject_id = ? AND grade = ?", subjectID, 3).Count(&gradeDistribution.Grade3)
	database.DB.Model(&models.Grade{}).Where("subject_id = ? AND grade = ?", subjectID, 2).Count(&gradeDistribution.Grade2)
	database.DB.Model(&models.Grade{}).Where("subject_id = ? AND grade = ?", subjectID, 1).Count(&gradeDistribution.Grade1)

	// Количество уроков
	var lessonsCount int64
	database.DB.Model(&models.Schedule{}).Where("subject_id = ?", subjectID).Count(&lessonsCount)

	c.JSON(http.StatusOK, gin.H{
		"subject":            subject,
		"average_grade":      avgGrade.Average,
		"total_grades":       avgGrade.Count,
		"grade_distribution": gradeDistribution,
		"lessons_count":      lessonsCount,
	})
}

// CompareClasses сравнивает классы по успеваемости
func (h *AnalyticsHandler) CompareClasses(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	type ClassComparison struct {
		ClassID            uint    `json:"class_id"`
		ClassName          string  `json:"class_name"`
		StudentsCount      int64   `json:"students_count"`
		AverageGrade       float64 `json:"average_grade"`
		AttendancePercent  float64 `json:"attendance_percent"`
	}

	var comparisons []ClassComparison

	// Получаем все классы школы
	var classes []models.Class
	database.DB.Where("school_id = ?", schoolID).Preload("Students").Find(&classes)

	for _, class := range classes {
		var comparison ClassComparison
		comparison.ClassID = class.ID
		comparison.ClassName = class.Name
		comparison.StudentsCount = int64(len(class.Students))

		// Средний балл класса
		var avgGrade struct {
			Average float64
		}
		database.DB.Table("grades").
			Joins("JOIN users ON users.id = grades.student_id").
			Joins("JOIN class_students ON class_students.user_id = users.id").
			Where("class_students.class_id = ?", class.ID).
			Select("AVG(grades.grade) as average").
			Scan(&avgGrade)
		comparison.AverageGrade = avgGrade.Average

		// Посещаемость класса
		var attendance struct {
			Total   int64
			Present int64
		}
		database.DB.Table("attendances").
			Joins("JOIN schedules ON schedules.id = attendances.schedule_id").
			Where("schedules.class_id = ?", class.ID).
			Count(&attendance.Total)

		database.DB.Table("attendances").
			Joins("JOIN schedules ON schedules.id = attendances.schedule_id").
			Where("schedules.class_id = ? AND attendances.status = ?", class.ID, "present").
			Count(&attendance.Present)

		if attendance.Total > 0 {
			comparison.AttendancePercent = float64(attendance.Present) / float64(attendance.Total) * 100
		}

		comparisons = append(comparisons, comparison)
	}

	c.JSON(http.StatusOK, gin.H{"comparisons": comparisons})
}
