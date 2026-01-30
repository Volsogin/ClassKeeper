package models

import (
	"time"

	"gorm.io/gorm"
)

// School представляет школу
type School struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null;size:255" json:"name"`
	Address   string         `gorm:"type:text" json:"address,omitempty"`
	Phone     string         `gorm:"size:20" json:"phone,omitempty"`
	Email     string         `gorm:"size:100" json:"email,omitempty"`
	LogoURL   string         `gorm:"size:500" json:"logo_url,omitempty"`
	AdminID   *uint          `json:"admin_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	Users   []User   `gorm:"foreignKey:SchoolID" json:"-"`
	Classes []Class  `gorm:"foreignKey:SchoolID" json:"-"`
}

// User представляет пользователя системы
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	SchoolID     uint           `gorm:"not null;index" json:"school_id"`
	Username     string         `gorm:"unique;not null;size:50" json:"username"`
	Email        string         `gorm:"unique;not null;size:100" json:"email"`
	PasswordHash string         `gorm:"not null;size:255" json:"-"`
	Role         string         `gorm:"not null;size:20" json:"role"` // admin, teacher, student, parent, starosta
	FirstName    string         `gorm:"size:100" json:"first_name,omitempty"`
	LastName     string         `gorm:"size:100" json:"last_name,omitempty"`
	MiddleName   string         `gorm:"size:100" json:"middle_name,omitempty"`
	AdminTitle   string         `gorm:"size:100" json:"admin_title,omitempty"` // Должность админа (завуч, старший учитель и т.д.)
	TeacherSubject string       `gorm:"size:100" json:"teacher_subject,omitempty"` // Предмет учителя
	AvatarURL    string         `gorm:"size:500" json:"avatar_url,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	School School `gorm:"foreignKey:SchoolID" json:"-"`
}

// Class представляет класс в школе
type Class struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	SchoolID           uint           `gorm:"not null;index" json:"school_id"`
	Name               string         `gorm:"not null;size:50" json:"name"` // "9А", "11Б"
	Year               string         `gorm:"not null;size:20" json:"year"` // учебный год "2025-2026"
	HomeroomTeacherID  *uint          `json:"homeroom_teacher_id,omitempty"`
	TeacherID          *uint          `gorm:"-" json:"-"` // Алиас для HomeroomTeacherID (для обратной совместимости в коде)
	StarostaID         *uint          `json:"starosta_id,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	School           School  `gorm:"foreignKey:SchoolID" json:"-"`
	HomeroomTeacher  *User   `gorm:"foreignKey:HomeroomTeacherID" json:"homeroom_teacher,omitempty"`
	Starosta         *User   `gorm:"foreignKey:StarostaID" json:"starosta,omitempty"`
	Students         []User  `gorm:"many2many:class_students;" json:"students,omitempty"`
}

// AfterFind hook для заполнения TeacherID
func (c *Class) AfterFind(tx *gorm.DB) error {
	c.TeacherID = c.HomeroomTeacherID
	return nil
}

// Subject представляет учебный предмет
type Subject struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	SchoolID    uint           `gorm:"not null;index" json:"school_id"`
	Name        string         `gorm:"not null;size:100" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	School   School `gorm:"foreignKey:SchoolID" json:"-"`
	Teachers []User `gorm:"many2many:teachers_subjects;" json:"teachers,omitempty"`
}

// Schedule представляет расписание урока
type Schedule struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ClassID      uint           `gorm:"not null;index" json:"class_id"`
	SubjectID    uint           `gorm:"not null;index" json:"subject_id"`
	TeacherID    *uint          `gorm:"index" json:"teacher_id,omitempty"` // Опционально
	DayOfWeek    string         `gorm:"not null;size:20" json:"day_of_week"` // Понедельник, Вторник...
	LessonNumber int            `gorm:"not null" json:"lesson_number"`
	StartTime    string         `gorm:"size:10" json:"start_time,omitempty"` // HH:MM формат
	EndTime      string         `gorm:"size:10" json:"end_time,omitempty"`   // HH:MM формат
	RoomNumber   string         `gorm:"size:50" json:"room_number,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	Class   Class   `gorm:"foreignKey:ClassID" json:"class,omitempty"`
	Subject Subject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Teacher *User   `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
}

// Attendance представляет посещаемость ученика
type Attendance struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	StudentID    uint           `gorm:"not null;index" json:"student_id"`
	ClassID      uint           `gorm:"not null;index" json:"class_id"`
	SubjectID    *uint          `gorm:"index" json:"subject_id,omitempty"`
	Date         time.Time      `gorm:"not null;type:date;index" json:"date"`
	LessonNumber *int           `json:"lesson_number,omitempty"`
	Status       string         `gorm:"not null;size:20" json:"status"` // present, absent, late, excused
	Comment      string         `gorm:"type:text" json:"comment,omitempty"`
	MarkedBy     *uint          `json:"marked_by,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	Student User     `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Class   Class    `gorm:"foreignKey:ClassID" json:"class,omitempty"`
	Subject *Subject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Marker  *User    `gorm:"foreignKey:MarkedBy" json:"marked_by_user,omitempty"`
}

// Grade представляет оценку ученика
type Grade struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	StudentID uint      `gorm:"not null;index" json:"student_id"`
	SubjectID uint      `gorm:"not null;index" json:"subject_id"`
	TeacherID uint      `gorm:"not null;index" json:"teacher_id"`
	Grade     int       `gorm:"not null" json:"grade"` // 2, 3, 4, 5
	GradeType string    `gorm:"size:20" json:"grade_type,omitempty"` // homework, test, exam, oral, final
	Date      time.Time `gorm:"not null;type:date" json:"date"`
	Comment   string    `gorm:"type:text" json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Связи
	Student User    `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Subject Subject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Teacher User    `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
}

// Homework представляет домашнее задание
type Homework struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ClassID      uint           `gorm:"not null;index" json:"class_id"`
	SubjectID    uint           `gorm:"not null;index" json:"subject_id"`
	TeacherID    uint           `gorm:"not null;index" json:"teacher_id"`
	Description  string         `gorm:"type:text;not null" json:"description"`
	AssignedDate time.Time      `gorm:"type:date;not null" json:"assigned_date"`
	DueDate      time.Time      `gorm:"type:date;not null" json:"due_date"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Связи
	Class   Class   `gorm:"foreignKey:ClassID" json:"class,omitempty"`
	Subject Subject `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Teacher User    `gorm:"foreignKey:TeacherID" json:"teacher,omitempty"`
}

// Announcement представляет объявление
type Announcement struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SchoolID      uint      `gorm:"not null;index" json:"school_id"`
	AuthorID      uint      `gorm:"not null;index" json:"author_id"`
	Title         string    `gorm:"not null;size:255" json:"title"`
	Content       string    `gorm:"not null;type:text" json:"content"`
	TargetRole    string    `gorm:"size:20" json:"target_role,omitempty"` // all, teachers, students, parents
	TargetClassID *uint     `json:"target_class_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`

	// Связи
	School      School `gorm:"foreignKey:SchoolID" json:"-"`
	Author      User   `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	TargetClass *Class `gorm:"foreignKey:TargetClassID" json:"target_class,omitempty"`
}

// ParentStudent связывает родителя с учеником
type ParentStudent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ParentID  uint      `gorm:"not null;index" json:"parent_id"`
	StudentID uint      `gorm:"not null;index" json:"student_id"`

	// Связи
	Parent    User      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Student   User      `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ClassStudent связывает класс с учеником (many2many таблица)
type ClassStudent struct {
	ClassID   uint `gorm:"primaryKey" json:"class_id"`
	StudentID uint `gorm:"primaryKey" json:"student_id"`

	// Связи
	Class   Class `gorm:"foreignKey:ClassID" json:"-"`
	Student User  `gorm:"foreignKey:StudentID" json:"-"`
}
