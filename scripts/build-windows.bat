@echo off
setlocal
cd /d "%~dp0\.."
if not exist release mkdir release
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=1.0.0" -o release\Pitchboard.exe .\cmd\pitchboard
if errorlevel 1 (
  echo Build failed.
  exit /b 1
)
echo Built release\Pitchboard.exe
echo Copy Pitchboard.exe anywhere. The schedule is saved in a PitchboardData folder beside it.
