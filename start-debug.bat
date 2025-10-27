@echo off
chcp 65001 >nul
cls

echo.
echo ========================================
echo   🎮 Project Abyss - Debug Mode
echo ========================================
echo.

set DEBUG=true

echo 🔍 调试模式启动...
echo.

go run cmd/server/main.go

pause

