@echo off
echo Building Go tool call tester...
go build -o test_tool_calls_go.exe test_tool_calls.go
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)
echo Build successful! Use test_tool_calls_go.exe to run tests.
pause