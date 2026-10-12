@echo off
rem Double-click to play the Habit alpha 0.5 demo tour (for recording a video).
cd /d "%~dp0"
if not exist node_modules (
  echo Installing the demo's one dependency...
  call npm install --no-audit --no-fund
)
node demo.mjs %*
pause
