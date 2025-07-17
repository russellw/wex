@echo off
echo Building Go tool call tester...
go build -o test_tool_calls_go.exe test_tool_calls.go
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Running Go tool call tester...
test_tool_calls_go.exe --ollama-url http://192.168.0.63:11434 --model qwen2.5-coder:14b
