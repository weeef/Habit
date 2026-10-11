@echo off
rem Double-click to update Habit from GitHub and start it.
cd /d "%~dp0"
echo Getting the latest version from GitHub...
git pull --ff-only
if errorlevel 1 echo Could not update (offline, or local changes). Starting the version you have.
echo.
echo Starting Habit. Your browser will open http://localhost:8080 in a few seconds.
echo Close this window to stop it.
start "" cmd /c "timeout /t 6 >nul & start http://localhost:8080"
go run ./cmd/habit
pause
