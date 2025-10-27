@echo off
chcp 65001 >nul
cls

echo.
echo ========================================
echo   🎮 Project Abyss - AI无限流跑团
echo ========================================
echo.

REM 检查是否已编译
if not exist "project-abyss.exe" (
    echo 📦 首次运行，正在编译项目...
    go build -o project-abyss.exe cmd/server/main.go
    if errorlevel 1 (
        echo.
        echo ❌ 编译失败，请检查Go环境是否正确安装
        pause
        exit /b 1
    )
    echo ✅ 编译完成！
    echo.
)

echo 🚀 正在启动服务器...
echo.
project-abyss.exe

pause
