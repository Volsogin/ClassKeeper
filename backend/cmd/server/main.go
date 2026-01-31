package main

import (
	"classkeeper/internal/config"
	"classkeeper/internal/database"
	"classkeeper/internal/handlers"
	"classkeeper/internal/middleware"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Подключаемся к базе данных
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Выполняем миграции
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Настраиваем Gin
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(middleware.CORSMiddleware())

	// Инициализируем handlers
	authHandler := handlers.NewAuthHandler(cfg)
	schoolHandler := handlers.NewSchoolHandler()
	userHandler := handlers.NewUserHandler()
	classHandler := handlers.NewClassHandler()
	subjectHandler := handlers.NewSubjectHandler()
	scheduleHandler := handlers.NewScheduleHandler()
	attendanceHandler := handlers.NewAttendanceHandler()
	gradeHandler := handlers.NewGradeHandler()
	homeworkHandler := handlers.NewHomeworkHandler()
	announcementHandler := handlers.NewAnnouncementHandler()
	analyticsHandler := handlers.NewAnalyticsHandler()
	exportHandler := handlers.NewExportHandler()
	parentHandler := handlers.NewParentHandler()
	settingsHandler := handlers.NewSettingsHandler()

	// API routes
	api := router.Group("/api")
	{
		// Публичные роуты (без аутентификации)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Создание школы (публично)
		api.POST("/schools", schoolHandler.CreateSchool)

		// Защищенные роуты (требуют аутентификации)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			// Текущий пользователь
			protected.GET("/auth/me", authHandler.Me)

			// Школы (только для админов)
			schools := protected.Group("/schools")
			schools.Use(middleware.RequireRole("admin"))
			{
				schools.GET("", schoolHandler.ListSchools)
				schools.GET("/:id", schoolHandler.GetSchool)
				schools.PUT("/:id", schoolHandler.UpdateSchool)
			}

			// Пользователи
			users := protected.Group("/users")
			{
				users.POST("", middleware.RequireRole("admin"), authHandler.Register)
				users.GET("", userHandler.ListUsers)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", middleware.RequireRole("admin"), userHandler.DeleteUser)
				users.PUT("/:id/password", userHandler.ChangePassword)
			}

			// Классы
			classes := protected.Group("/classes")
			{
				classes.POST("", middleware.RequireRole("admin"), classHandler.CreateClass)
				classes.GET("", classHandler.ListClasses)
				classes.GET("/:id", classHandler.GetClass)
				classes.PUT("/:id", middleware.RequireRole("admin"), classHandler.UpdateClass)
				classes.DELETE("/:id", middleware.RequireRole("admin"), classHandler.DeleteClass)
				classes.POST("/:id/students", middleware.RequireRole("admin"), classHandler.AddStudents)
				classes.DELETE("/:id/students/:student_id", middleware.RequireRole("admin"), classHandler.RemoveStudent)
			}

			// Предметы
			subjects := protected.Group("/subjects")
			{
				subjects.POST("", middleware.RequireRole("admin"), subjectHandler.CreateSubject)
				subjects.GET("", subjectHandler.ListSubjects)
				subjects.GET("/:id", subjectHandler.GetSubject)
				subjects.PUT("/:id", middleware.RequireRole("admin"), subjectHandler.UpdateSubject)
				subjects.DELETE("/:id", middleware.RequireRole("admin"), subjectHandler.DeleteSubject)
				subjects.POST("/:id/teachers", middleware.RequireRole("admin"), subjectHandler.AssignTeachers)
				subjects.DELETE("/:id/teachers/:teacher_id", middleware.RequireRole("admin"), subjectHandler.RemoveTeacher)
			}

			// Расписание
			schedules := protected.Group("/schedules")
			{
				schedules.POST("", middleware.RequireRole("admin", "teacher"), scheduleHandler.CreateSchedule)
				schedules.GET("", scheduleHandler.ListSchedules)
				schedules.GET("/:id", scheduleHandler.GetSchedule)
				schedules.PUT("/:id", middleware.RequireRole("admin", "teacher"), scheduleHandler.UpdateSchedule)
				schedules.DELETE("/:id", middleware.RequireRole("admin"), scheduleHandler.DeleteSchedule)
				schedules.GET("/class/:id", scheduleHandler.GetClassSchedule)
			}

			// Посещаемость
			attendance := protected.Group("/attendance")
			{
				attendance.POST("", middleware.RequireRole("admin", "teacher", "starosta"), attendanceHandler.MarkAttendance)
				attendance.POST("/bulk", middleware.RequireRole("admin", "teacher", "starosta"), attendanceHandler.BulkMarkAttendance)
				attendance.GET("", attendanceHandler.GetAttendance)
				attendance.GET("/student/:id/stats", attendanceHandler.GetStudentStats)
				attendance.DELETE("/:id", middleware.RequireRole("admin"), attendanceHandler.DeleteAttendance)
			}

			// Оценки
			grades := protected.Group("/grades")
			{
				grades.POST("", middleware.RequireRole("admin", "teacher"), gradeHandler.CreateGrade)
				grades.GET("", gradeHandler.ListGrades)
				grades.GET("/:id", gradeHandler.GetGrade)
				grades.PUT("/:id", middleware.RequireRole("admin", "teacher"), gradeHandler.UpdateGrade)
				grades.DELETE("/:id", middleware.RequireRole("admin", "teacher"), gradeHandler.DeleteGrade)
				grades.GET("/student/:id/average", gradeHandler.GetStudentAverage)
				grades.GET("/class/:id/journal", gradeHandler.GetClassJournal)
			}

			// Домашние задания
			homework := protected.Group("/homework")
			{
				homework.POST("", middleware.RequireRole("admin", "teacher"), homeworkHandler.CreateHomework)
				homework.GET("", homeworkHandler.ListHomework)
				homework.GET("/:id", homeworkHandler.GetHomework)
				homework.PUT("/:id", middleware.RequireRole("admin", "teacher"), homeworkHandler.UpdateHomework)
				homework.DELETE("/:id", middleware.RequireRole("admin", "teacher"), homeworkHandler.DeleteHomework)
				homework.GET("/class/:id/upcoming", homeworkHandler.GetUpcomingHomework)
				homework.GET("/class/:id/overdue", homeworkHandler.GetOverdueHomework)
			}

			// Объявления
			announcements := protected.Group("/announcements")
			{
				announcements.POST("", middleware.RequireRole("admin", "teacher"), announcementHandler.CreateAnnouncement)
				announcements.GET("", announcementHandler.ListAnnouncements)
				announcements.GET("/my", announcementHandler.GetMyAnnouncements)
				announcements.GET("/class/:id", announcementHandler.GetClassAnnouncements)
				announcements.GET("/:id", announcementHandler.GetAnnouncement)
				announcements.PUT("/:id", middleware.RequireRole("admin", "teacher"), announcementHandler.UpdateAnnouncement)
				announcements.DELETE("/:id", middleware.RequireRole("admin", "teacher"), announcementHandler.DeleteAnnouncement)
			}

			// Аналитика
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/school", analyticsHandler.GetSchoolStats)
				analytics.GET("/class/:id", analyticsHandler.GetClassStats)
				analytics.GET("/teacher/:id", analyticsHandler.GetTeacherStats)
				analytics.GET("/subject/:id", analyticsHandler.GetSubjectStats)
				analytics.GET("/attendance-report", analyticsHandler.GetAttendanceReport)
				analytics.GET("/grades-report", analyticsHandler.GetGradesReport)
				analytics.GET("/compare-classes", analyticsHandler.CompareClasses)
			}

			// Экспорт данных
			export := protected.Group("/export")
			{
				export.GET("/class/:id/grades", exportHandler.ExportClassGrades)
				export.GET("/class/:id/attendance", exportHandler.ExportClassAttendance)
				export.GET("/student/:id/report", exportHandler.ExportStudentReport)
				export.GET("/school/report", exportHandler.ExportSchoolReport)
			}

			// Родители
			parents := protected.Group("/parents")
			{
				parents.POST("/link", middleware.RequireRole("admin"), parentHandler.LinkParentToStudent)
				parents.DELETE("/:parent_id/students/:student_id", middleware.RequireRole("admin"), parentHandler.UnlinkParentFromStudent)
				parents.GET("/children", parentHandler.GetParentChildren)
				parents.GET("/:id/children", middleware.RequireRole("admin"), parentHandler.GetParentChildren)
				parents.GET("/students/:id/parents", parentHandler.GetStudentParents)
				parents.GET("/child/:id/grades", parentHandler.GetChildGrades)
				parents.GET("/child/:id/attendance", parentHandler.GetChildAttendance)
				parents.GET("/child/:id/homework", parentHandler.GetChildHomework)
			}

			// Связи родителей и детей
			parentStudentHandler := handlers.NewParentStudentHandler()
			parentStudentLinks := protected.Group("/parent-student-links")
			{
				parentStudentLinks.POST("", middleware.RequireRole("admin"), parentStudentHandler.CreateLink)
				parentStudentLinks.GET("", parentStudentHandler.ListLinks)
				parentStudentLinks.DELETE("/:id", middleware.RequireRole("admin"), parentStudentHandler.DeleteLink)
				parentStudentLinks.GET("/parent/:parent_id/students", parentStudentHandler.GetStudentsByParent)
				parentStudentLinks.GET("/student/:student_id/parents", parentStudentHandler.GetParentsByStudent)
			}

			// Настройки
			settings := protected.Group("/settings")
			{
				settings.GET("/school", settingsHandler.GetSchoolSettings)
				settings.PUT("/school", middleware.RequireRole("admin"), settingsHandler.UpdateSchoolSettings)
				settings.GET("/system", settingsHandler.GetSystemInfo)
				settings.GET("/backup", middleware.RequireRole("admin"), settingsHandler.BackupDatabase)
				settings.GET("/audit", middleware.RequireRole("admin"), settingsHandler.GetAuditLog)
			}
		}
	}

	// Раздача статических файлов (frontend)
	router.Static("/static", "../../../frontend")
	router.Static("/css", "../../../frontend/css")
	router.Static("/js", "../../../frontend/js")
	router.Static("/pages", "../../../frontend/pages")

	// ═══════════════════════════════════════════════════════════
	// DEMO MODE - Проверяем переменную окружения DEMO_MODE
	// ═══════════════════════════════════════════════════════════
	demoMode := os.Getenv("DEMO_MODE") == "false"

	if demoMode {
		log.Println("🎭 DEMO MODE ENABLED - Using demo.html as landing page")
		router.StaticFile("/", "../../../frontend/demo.html")
		router.StaticFile("/login", "../../../frontend/pages/index.html")
	} else {
		log.Println("🏫 PRODUCTION MODE - Using index.html as landing page")
		router.StaticFile("/", "../../../frontend/pages/index.html")
	}

	// Запускаем сервер
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}