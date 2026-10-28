@echo off
setlocal
cd /d "%~dp0"
if exist "%~dp0Pitchboard.exe" (
  start "" "%~dp0Pitchboard.exe"
  exit /b 0
)
if exist "%~dp0release\Pitchboard.exe" (
  start "" "%~dp0release\Pitchboard.exe"
  exit /b 0
)
where go >nul 2>&1
if %errorlevel%==0 (
  go run ./cmd/pitchboard
  exit /b %errorlevel%
)
echo Pitchboard.exe was not found, and Go is not installed.
echo Build the portable program with scripts\build-windows.bat
pause
