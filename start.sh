#!/bin/bash

echo "========================================"
echo "  ClassKeeper 2.0 - Запуск сервера"
echo "========================================"
echo ""

cd backend/cmd/server

echo "Проверка зависимостей Go..."
go mod download
if [ $? -ne 0 ]; then
    echo "Ошибка загрузки зависимостей!"
    exit 1
fi

echo ""
echo "Запуск сервера..."
echo ""
echo "Сервер будет доступен по адресу: http://localhost:8080"
echo ""
echo "Для остановки нажмите Ctrl+C"
echo ""
echo "========================================"
echo ""

go run main.go
