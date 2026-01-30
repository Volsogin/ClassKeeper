package handlers

import (
	"classkeeper/internal/database"
	"classkeeper/internal/models"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct{}

func NewExportHandler() *ExportHandler {
	return &ExportHandler{}
}

// ExportClassGrades экспортирует оценки класса в CSV
func (h *ExportHandler) ExportClassGrades(c *gin.Context) {
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

	// Получаем оценки
	var grades []models.Grade
	studentIDs := make([]uint, len(class.Students))
	for i, s := range class.Students {
		studentIDs[i] = s.ID
	}

	database.DB.Where("student_id IN ?", studentIDs).
		Preload("Student").
		Preload("Subject").
		Preload("Teacher").
		Order("date DESC").
		Find(&grades)

	// Создаём CSV
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=grades_class_%s.csv", class.Name))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Заголовки
	writer.Write([]string{"Дата", "Ученик", "Предмет", "Оценка", "Тип", "Учитель", "Комментарий"})

	// Данные
	for _, grade := range grades {
		writer.Write([]string{
			grade.Date.Format("2006-01-02"),
			fmt.Sprintf("%s %s", grade.Student.FirstName, grade.Student.LastName),
			grade.Subject.Name,
			strconv.Itoa(grade.Grade),
			grade.GradeType,
			fmt.Sprintf("%s %s", grade.Teacher.FirstName, grade.Teacher.LastName),
			grade.Comment,
		})
	}
}

// ExportClassAttendance экспортирует посещаемость класса в CSV
func (h *ExportHandler) ExportClassAttendance(c *gin.Context) {
	classID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	schoolID, _ := c.Get("school_id")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = time.Now().Format("2006-01-02")
	}

	// Проверяем класс
	var class models.Class
	if err := database.DB.Where("id = ? AND school_id = ?", classID, schoolID).
		First(&class).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	// Получаем посещаемость
	var attendance []models.Attendance
	database.DB.Joins("JOIN schedules ON schedules.id = attendances.schedule_id").
		Where("schedules.class_id = ? AND attendances.date BETWEEN ? AND ?", classID, dateFrom, dateTo).
		Preload("Student").
		Preload("Schedule.Subject").
		Preload("Marker").
		Order("date DESC").
		Find(&attendance)

	// Создаём CSV
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=attendance_class_%s.csv", class.Name))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// BOM для правильной кодировки в Excel
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	// Заголовки
	writer.Write([]string{"Дата", "Ученик", "Предмет", "Статус", "Отметил", "Комментарий"})

	// Данные
	for _, a := range attendance {
		markerName := ""
		if a.Marker != nil {
			markerName = fmt.Sprintf("%s %s", a.Marker.FirstName, a.Marker.LastName)
		}

		subjectName := "-"
		if a.Subject != nil {
			subjectName = a.Subject.Name
		}

		writer.Write([]string{
			a.Date.Format("2006-01-02"),
			fmt.Sprintf("%s %s", a.Student.FirstName, a.Student.LastName),
			subjectName,
			a.Status,
			markerName,
			a.Comment,
		})
	}
}

// ExportStudentReport экспортирует полный отчёт об ученике
func (h *ExportHandler) ExportStudentReport(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}

	schoolID, _ := c.Get("school_id")

	// Проверяем ученика
	var student models.User
	if err := database.DB.Where("id = ? AND school_id = ?", studentID, schoolID).
		First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=report_student_%s_%s.csv", 
		student.LastName, student.FirstName))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// BOM
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	// Информация об ученике
	writer.Write([]string{"ОТЧЁТ ОБ УЧЕНИКЕ"})
	writer.Write([]string{"ФИО", fmt.Sprintf("%s %s %s", student.LastName, student.FirstName, student.MiddleName)})
	writer.Write([]string{""})

	// Оценки
	writer.Write([]string{"ОЦЕНКИ"})
	writer.Write([]string{"Дата", "Предмет", "Оценка", "Тип", "Учитель"})

	var grades []models.Grade
	database.DB.Where("student_id = ?", studentID).
		Preload("Subject").
		Preload("Teacher").
		Order("date DESC").
		Limit(100).
		Find(&grades)

	for _, grade := range grades {
		writer.Write([]string{
			grade.Date.Format("2006-01-02"),
			grade.Subject.Name,
			strconv.Itoa(grade.Grade),
			grade.GradeType,
			fmt.Sprintf("%s %s", grade.Teacher.FirstName, grade.Teacher.LastName),
		})
	}

	// Средний балл по предметам
	writer.Write([]string{""})
	writer.Write([]string{"СРЕДНИЙ БАЛЛ ПО ПРЕДМЕТАМ"})
	writer.Write([]string{"Предмет", "Средний балл", "Количество оценок"})

	var subjectAverages []struct {
		SubjectName string
		Average     float64
		Count       int64
	}

	database.DB.Table("grades").
		Select("subjects.name as subject_name, AVG(grades.grade) as average, COUNT(*) as count").
		Joins("JOIN subjects ON subjects.id = grades.subject_id").
		Where("grades.student_id = ?", studentID).
		Group("subjects.id, subjects.name").
		Scan(&subjectAverages)

	for _, avg := range subjectAverages {
		writer.Write([]string{
			avg.SubjectName,
			fmt.Sprintf("%.2f", avg.Average),
			strconv.FormatInt(avg.Count, 10),
		})
	}

	// Посещаемость
	writer.Write([]string{""})
	writer.Write([]string{"ПОСЕЩАЕМОСТЬ (последние 100 записей)"})
	writer.Write([]string{"Дата", "Предмет", "Статус"})

	var attendance []models.Attendance
	database.DB.Where("student_id = ?", studentID).
		Preload("Subject").
		Order("date DESC").
		Limit(100).
		Find(&attendance)

	for _, a := range attendance {
		subjectName := "-"
		if a.Subject != nil {
			subjectName = a.Subject.Name
		}
		writer.Write([]string{
			a.Date.Format("2006-01-02"),
			subjectName,
			a.Status,
		})
	}

	// Статистика посещаемости
	writer.Write([]string{""})
	writer.Write([]string{"СТАТИСТИКА ПОСЕЩАЕМОСТИ"})

	var stats struct {
		Total   int64
		Present int64
		Absent  int64
		Late    int64
		Sick    int64
		Excused int64
	}

	database.DB.Model(&models.Attendance{}).Where("student_id = ?", studentID).Count(&stats.Total)
	database.DB.Model(&models.Attendance{}).Where("student_id = ? AND status = ?", studentID, "present").Count(&stats.Present)
	database.DB.Model(&models.Attendance{}).Where("student_id = ? AND status = ?", studentID, "absent").Count(&stats.Absent)
	database.DB.Model(&models.Attendance{}).Where("student_id = ? AND status = ?", studentID, "late").Count(&stats.Late)
	database.DB.Model(&models.Attendance{}).Where("student_id = ? AND status = ?", studentID, "sick").Count(&stats.Sick)
	database.DB.Model(&models.Attendance{}).Where("student_id = ? AND status = ?", studentID, "excused").Count(&stats.Excused)

	var percentage float64
	if stats.Total > 0 {
		percentage = float64(stats.Present) / float64(stats.Total) * 100
	}

	writer.Write([]string{"Всего отметок", strconv.FormatInt(stats.Total, 10)})
	writer.Write([]string{"Присутствовал", strconv.FormatInt(stats.Present, 10)})
	writer.Write([]string{"Отсутствовал", strconv.FormatInt(stats.Absent, 10)})
	writer.Write([]string{"Опоздал", strconv.FormatInt(stats.Late, 10)})
	writer.Write([]string{"Болел", strconv.FormatInt(stats.Sick, 10)})
	writer.Write([]string{"По уважительной причине", strconv.FormatInt(stats.Excused, 10)})
	writer.Write([]string{"Процент посещаемости", fmt.Sprintf("%.2f%%", percentage)})
}

// ExportSchoolReport экспортирует общий отчёт по школе
func (h *ExportHandler) ExportSchoolReport(c *gin.Context) {
	schoolID, _ := c.Get("school_id")

	// Получаем школу
	var school models.School
	database.DB.First(&school, schoolID)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=school_report_%s.csv", 
		time.Now().Format("2006-01-02")))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// BOM
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	// Заголовок
	writer.Write([]string{"ОТЧЁТ ПО ШКОЛЕ"})
	writer.Write([]string{"Школа", school.Name})
	writer.Write([]string{"Дата отчёта", time.Now().Format("2006-01-02")})
	writer.Write([]string{""})

	// Общая статистика
	writer.Write([]string{"ОБЩАЯ СТАТИСТИКА"})

	var studentsCount, teachersCount, classesCount int64
	database.DB.Model(&models.User{}).Where("school_id = ? AND (role = ? OR role = ?)", 
		schoolID, "student", "starosta").Count(&studentsCount)
	database.DB.Model(&models.User{}).Where("school_id = ? AND role = ?", 
		schoolID, "teacher").Count(&teachersCount)
	database.DB.Model(&models.Class{}).Where("school_id = ?", schoolID).Count(&classesCount)

	writer.Write([]string{"Учеников", strconv.FormatInt(studentsCount, 10)})
	writer.Write([]string{"Учителей", strconv.FormatInt(teachersCount, 10)})
	writer.Write([]string{"Классов", strconv.FormatInt(classesCount, 10)})
	writer.Write([]string{""})

	// Статистика по классам
	writer.Write([]string{"СТАТИСТИКА ПО КЛАССАМ"})
	writer.Write([]string{"Класс", "Учеников", "Средний балл", "Посещаемость %"})

	var classes []models.Class
	database.DB.Where("school_id = ?", schoolID).Preload("Students").Find(&classes)

	for _, class := range classes {
		var avgGrade struct {
			Average float64
		}
		database.DB.Table("grades").
			Joins("JOIN users ON users.id = grades.student_id").
			Joins("JOIN class_students ON class_students.user_id = users.id").
			Where("class_students.class_id = ?", class.ID).
			Select("AVG(grades.grade) as average").
			Scan(&avgGrade)

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

		var attendancePercent float64
		if attendance.Total > 0 {
			attendancePercent = float64(attendance.Present) / float64(attendance.Total) * 100
		}

		writer.Write([]string{
			class.Name,
			strconv.Itoa(len(class.Students)),
			fmt.Sprintf("%.2f", avgGrade.Average),
			fmt.Sprintf("%.2f", attendancePercent),
		})
	}
}
