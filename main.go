package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Engine struct {
	ollamaURL string
	model     string
	workspace string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	Stream   bool      `json:"stream"`
}

type ChatResponse struct {
	Message struct {
		Role      string     `json:"role"`
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done bool `json:"done"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
}

type Model struct {
	Name string `json:"name"`
}

type ModelsResponse struct {
	Models []Model `json:"models"`
}

func NewEngine(ollamaURL, model, workspace string) (*Engine, error) {
	engine := &Engine{
		ollamaURL: ollamaURL,
		workspace: workspace,
	}

	if model == "" {
		firstModel, err := engine.getFirstAvailableModel()
		if err != nil {
			return nil, fmt.Errorf("failed to get available model: %v", err)
		}
		engine.model = firstModel
	} else {
		engine.model = model
	}

	return engine, nil
}

func (e *Engine) getFirstAvailableModel() (string, error) {
	resp, err := http.Get(e.ollamaURL + "/api/tags")
	if err != nil {
		return "", fmt.Errorf("failed to get models: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return "", fmt.Errorf("failed to decode models response: %v", err)
	}

	if len(modelsResp.Models) == 0 {
		return "", fmt.Errorf("no models available on Ollama server")
	}

	return modelsResp.Models[0].Name, nil
}

func (e *Engine) getTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "read_file",
				Description: "Read the contents of a file",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file to read",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "write_file",
				Description: "Write content to a file",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file to write",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Content to write to the file",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "run_command",
				Description: "Execute a shell command",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Shell command to execute",
						},
						"timeout": map[string]interface{}{
							"type":        "number",
							"description": "Timeout in seconds (optional, default 30)",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}

func (e *Engine) callTool(toolCall ToolCall) (string, error) {
	switch toolCall.Function.Name {
	case "read_file":
		return e.readFile(toolCall.Function.Arguments)
	case "write_file":
		return e.writeFile(toolCall.Function.Arguments)
	case "run_command":
		return e.runCommand(toolCall.Function.Arguments)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}
}

func (e *Engine) readFile(args json.RawMessage) (string, error) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	fullPath := filepath.Join(e.workspace, params.Path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	return string(content), nil
}

func (e *Engine) writeFile(args json.RawMessage) (string, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	fullPath := filepath.Join(e.workspace, params.Path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(fullPath, []byte(params.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}
	return fmt.Sprintf("Successfully wrote to %s", params.Path), nil
}

func (e *Engine) runCommand(args json.RawMessage) (string, error) {
	var params struct {
		Command string  `json:"command"`
		Timeout float64 `json:"timeout"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	if params.Timeout == 0 {
		params.Timeout = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(params.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)
	cmd.Dir = e.workspace

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %v\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

func (e *Engine) sendChatRequest(messages []Message) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:    e.model,
		Messages: messages,
		Tools:    e.getTools(),
		Stream:   false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(e.ollamaURL+"/api/chat", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &chatResp, nil
}

func (e *Engine) ProcessRequest(userMessage string) error {
	messages := []Message{
		{Role: "user", Content: userMessage},
	}

	for {
		resp, err := e.sendChatRequest(messages)
		if err != nil {
			return fmt.Errorf("chat request failed: %v", err)
		}

		messages = append(messages, Message{
			Role:    resp.Message.Role,
			Content: resp.Message.Content,
		})

		if resp.Message.Content != "" {
			fmt.Printf("Assistant: %s\n", resp.Message.Content)
		}

		if len(resp.Message.ToolCalls) == 0 {
			break
		}

		for _, toolCall := range resp.Message.ToolCalls {
			fmt.Printf("Executing tool: %s\n", toolCall.Function.Name)
			
			result, err := e.callTool(toolCall)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			messages = append(messages, Message{
				Role:    "tool",
				Content: result,
			})

			fmt.Printf("Tool result: %s\n", result)
		}
	}

	return nil
}

func main() {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://192.168.0.63:11434"
	}

	model := os.Getenv("OLLAMA_MODEL")

	workspace := os.Getenv("WORKSPACE")
	if workspace == "" {
		workspace = "/workspace"
	}

	engine, err := NewEngine(ollamaURL, model, workspace)
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	fmt.Printf("Using model: %s\n", engine.model)

	if len(os.Args) < 2 {
		log.Fatal("Usage: wex <message>")
	}

	userMessage := strings.Join(os.Args[1:], " ")
	if err := engine.ProcessRequest(userMessage); err != nil {
		log.Fatalf("Error processing request: %v", err)
	}
}