@echo off
echo Скачиваем зависимости...
go mod download golang.org/x/crypto
go mod download gorm.io/driver/sqlite
go mod download gorm.io/gorm
echo.
echo Готово! Теперь запусти: go run fill_db.go
pause
