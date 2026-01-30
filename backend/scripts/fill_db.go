package main

import (
	"fmt"
	"log"
	"math/rand"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
)

type School struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

type User struct {
	ID             uint   `gorm:"primaryKey"`
	SchoolID       uint   `gorm:"not null"`
	Username       string `gorm:"unique;not null"`
	Email          string `gorm:"unique;not null"`
	PasswordHash   string `gorm:"not null"`
	Role           string `gorm:"not null"`
	FirstName      string
	LastName       string
	MiddleName     string
	TeacherSubject string
}

type Subject struct {
	ID          uint   `gorm:"primaryKey"`
	SchoolID    uint   `gorm:"not null"`
	Name        string `gorm:"not null"`
	Description string
}

type Class struct {
	ID                uint   `gorm:"primaryKey"`
	SchoolID          uint   `gorm:"not null"`
	Name              string `gorm:"not null"`
	Year              string `gorm:"not null"`
	HomeroomTeacherID *uint
}

type ClassStudent struct {
	ClassID   uint `gorm:"primaryKey"`
	StudentID uint `gorm:"primaryKey"`
}

type ParentStudent struct {
	ID        uint `gorm:"primaryKey"`
	ParentID  uint `gorm:"not null"`
	StudentID uint `gorm:"not null"`
}

type Schedule struct {
	ID           uint   `gorm:"primaryKey"`
	SchoolID     uint   `gorm:"not null"`
	ClassID      uint   `gorm:"not null"`
	SubjectID    uint   `gorm:"not null"`
	DayOfWeek    string `gorm:"not null"`
	LessonNumber int    `gorm:"not null"`
	StartTime    string
	EndTime      string
	RoomNumber   string
}

var (
	firstNames = []string{"Александр", "Дмитрий", "Максим", "Иван", "Артём", "Михаил", "Даниил", "Егор", "Никита", "Кирилл",
		"Анна", "Мария", "Елена", "Ольга", "Наталья", "Екатерина", "Татьяна", "Ирина", "Светлана", "Людмила"}
	lastNames = []string{"Иванов", "Петров", "Сидоров", "Смирнов", "Кузнецов", "Попов", "Васильев", "Соколов", "Михайлов", "Новиков",
		"Иванова", "Петрова", "Сидорова", "Смирнова", "Кузнецова", "Попова", "Васильева", "Соколова", "Михайлова", "Новикова"}
	subjects  = []string{"Математика", "Русский язык", "Литература", "Английский язык", "История", "Обществознание", "Физика", "Химия", "Биология", "География", "Информатика", "Физкультура"}
	classes   = []string{"9А", "9Б", "10А", "10Б", "11А"}
	days      = []string{"Понедельник", "Вторник", "Среда", "Четверг", "Пятница"}
	times     = [][2]string{
		{"08:00", "08:45"}, {"08:55", "09:40"}, {"09:50", "10:35"},
		{"10:55", "11:40"}, {"11:50", "12:35"}, {"12:45", "13:30"},
	}
)

func main() {
	db, err := gorm.Open(sqlite.Open("classkeeper.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Получаем школу
	var school School
	db.First(&school)
	if school.ID == 0 {
		log.Fatal("Школа не найдена! Сначала создайте школу через интерфейс.")
	}

	schoolID := school.ID
	fmt.Printf("Заполняем школу: %s (ID: %d)\n", school.Name, schoolID)

	// Создаём предметы
	fmt.Println("\n📚 Создаём предметы...")
	var subjectIDs []uint
	for _, subjectName := range subjects {
		var existingSubject Subject
		db.Where("name = ? AND school_id = ?", subjectName, schoolID).First(&existingSubject)
		if existingSubject.ID == 0 {
			subject := Subject{
				SchoolID:    schoolID,
				Name:        subjectName,
				Description: fmt.Sprintf("Предмет %s", subjectName),
			}
			db.Create(&subject)
			subjectIDs = append(subjectIDs, subject.ID)
			fmt.Printf("  ✅ %s\n", subjectName)
		} else {
			subjectIDs = append(subjectIDs, existingSubject.ID)
		}
	}

	// Создаём учителей
	fmt.Println("\n👨‍🏫 Создаём учителей...")
	var teacherIDs []uint
	password, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	
	for i, subjectName := range subjects {
		teacher := User{
			SchoolID:       schoolID,
			Username:       fmt.Sprintf("teacher%d", i+1),
			Email:          fmt.Sprintf("teacher%d@school.ru", i+1),
			PasswordHash:   string(password),
			Role:           "teacher",
			FirstName:      firstNames[rand.Intn(len(firstNames))],
			LastName:       lastNames[rand.Intn(len(lastNames))],
			MiddleName:     "Викторович",
			TeacherSubject: subjectName,
		}
		db.Create(&teacher)
		teacherIDs = append(teacherIDs, teacher.ID)
		fmt.Printf("  ✅ %s %s - %s\n", teacher.LastName, teacher.FirstName, subjectName)
	}

	// Создаём классы
	fmt.Println("\n🎓 Создаём классы...")
	var classIDs []uint
	for i, className := range classes {
		class := Class{
			SchoolID:          schoolID,
			Name:              className,
			Year:              "2025-2026",
			HomeroomTeacherID: &teacherIDs[i%len(teacherIDs)],
		}
		db.Create(&class)
		classIDs = append(classIDs, class.ID)
		fmt.Printf("  ✅ %s\n", className)
	}

	// Создаём учеников (20 на класс = 100 учеников)
	fmt.Println("\n👨‍🎓 Создаём учеников...")
	var studentIDs []uint
	studentCounter := 1
	
	for _, classID := range classIDs {
		for j := 0; j < 20; j++ {
			student := User{
				SchoolID:     schoolID,
				Username:     fmt.Sprintf("student%d", studentCounter),
				Email:        fmt.Sprintf("student%d@school.ru", studentCounter),
				PasswordHash: string(password),
				Role:         "student",
				FirstName:    firstNames[rand.Intn(len(firstNames))],
				LastName:     lastNames[rand.Intn(len(lastNames))],
				MiddleName:   "Александрович",
			}
			db.Create(&student)
			studentIDs = append(studentIDs, student.ID)

			// Добавляем в класс
			db.Create(&ClassStudent{
				ClassID:   classID,
				StudentID: student.ID,
			})
			studentCounter++
		}
	}
	fmt.Printf("  ✅ Создано %d учеников\n", len(studentIDs))

	// Создаём родителей (по 1 родителю на ученика)
	fmt.Println("\n👨‍👩‍👧 Создаём родителей и связываем с детьми...")
	for i, studentID := range studentIDs {
		parent := User{
			SchoolID:     schoolID,
			Username:     fmt.Sprintf("parent%d", i+1),
			Email:        fmt.Sprintf("parent%d@school.ru", i+1),
			PasswordHash: string(password),
			Role:         "parent",
			FirstName:    firstNames[rand.Intn(len(firstNames))],
			LastName:     lastNames[rand.Intn(len(lastNames))],
			MiddleName:   "Петрович",
		}
		db.Create(&parent)

		// Связываем с ребёнком
		db.Create(&ParentStudent{
			ParentID:  parent.ID,
			StudentID: studentID,
		})
	}
	fmt.Printf("  ✅ Создано %d родителей\n", len(studentIDs))

	// Создаём расписание для каждого класса
	fmt.Println("\n📅 Создаём расписание...")
	scheduleCounter := 0
	
	for _, classID := range classIDs {
		lessonNum := 1
		for _, day := range days {
			// 6 уроков в день
			for i := 0; i < 6; i++ {
				if i >= len(subjectIDs) {
					break
				}
				
				schedule := Schedule{
					SchoolID:     schoolID,
					ClassID:      classID,
					SubjectID:    subjectIDs[i],
					DayOfWeek:    day,
					LessonNumber: lessonNum,
					StartTime:    times[i][0],
					EndTime:      times[i][1],
					RoomNumber:   fmt.Sprintf("%d", 200+rand.Intn(20)),
				}
				db.Create(&schedule)
				scheduleCounter++
				lessonNum++
			}
			lessonNum = 1
		}
	}
	fmt.Printf("  ✅ Создано %d уроков расписания\n", scheduleCounter)

	fmt.Println("\n🎉 ГОТОВО! База данных заполнена!")
	fmt.Println("\n📊 Итого:")
	fmt.Printf("  - Предметов: %d\n", len(subjectIDs))
	fmt.Printf("  - Учителей: %d\n", len(teacherIDs))
	fmt.Printf("  - Классов: %d\n", len(classIDs))
	fmt.Printf("  - Учеников: %d\n", len(studentIDs))
	fmt.Printf("  - Родителей: %d\n", len(studentIDs))
	fmt.Printf("  - Расписание: %d уроков\n", scheduleCounter)
	fmt.Println("\n🔐 Пароль для всех пользователей: password123")
}
