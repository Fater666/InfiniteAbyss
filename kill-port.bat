@echo off
chcp 65001 >nul
cls

echo.
echo ========================================
echo   🔧 关闭占用8080端口的进程
echo ========================================
echo.

echo 🔍 正在查找占用8080端口的进程...
echo.

for /f "tokens=5" %%a in ('netstat -ano ^| findstr :8080 ^| findstr LISTENING') do (
    set PID=%%a
)

if not defined PID (
    echo ✅ 8080端口没有被占用
    echo.
    pause
    exit /b 0
)

echo 找到进程 PID: %PID%
echo.

tasklist /FI "PID eq %PID%"
echo.

echo 正在关闭进程...
taskkill /F /PID %PID%

if %errorlevel% == 0 (
    echo.
    echo ✅ 成功关闭进程！
    echo 现在可以启动服务器了。
) else (
    echo.
    echo ❌ 关闭失败，请手动关闭或以管理员身份运行此脚本
)

echo.
pause

