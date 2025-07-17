package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// TestStatus represents the result status of a test
type TestStatus string

const (
	TestStatusPass    TestStatus = "PASS"
	TestStatusFail    TestStatus = "FAIL"
	TestStatusPartial TestStatus = "PARTIAL"
	TestStatusSkip    TestStatus = "SKIP"
)

// TestCase represents a single test case
type TestCase struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SystemPrompt    string   `json:"system_prompt"`
	UserMessage     string   `json:"user_message"`
	ExpectedTools   []string `json:"expected_tools"`
	SuccessCriteria string   `json:"success_criteria"`
	Timeout         int      `json:"timeout"`
}

// ToolCallResult represents the result of a tool call execution
type ToolCallResult struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

// TestResult represents the result of a test execution
type TestResult struct {
	TestName        string           `json:"test_name"`
	Result          TestStatus       `json:"result"`
	ToolCalls       []ToolCallResult `json:"tool_calls"`
	ResponseContent string           `json:"response_content"`
	Duration        float64          `json:"duration"`
	Notes           string           `json:"notes,omitempty"`
}

// Tool represents a function tool definition
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat API request
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools"`
	Stream   bool      `json:"stream"`
}

// ChatResponse represents a chat API response
type ChatResponse struct {
	Message struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done bool `json:"done"`
}

// LLMToolCallTester is the main tester struct
type LLMToolCallTester struct {
	OllamaURL string
	Model     string
	Tools     []Tool
}

// NewLLMToolCallTester creates a new tester instance
func NewLLMToolCallTester(ollamaURL, model string) *LLMToolCallTester {
	return &LLMToolCallTester{
		OllamaURL: strings.TrimRight(ollamaURL, "/"),
		Model:     model,
		Tools:     getTestTools(),
	}
}

// getTestTools returns the test tools available to the LLM
func getTestTools() []Tool {
	return []Tool{
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
		{
			Type: "function",
			Function: Function{
				Name:        "calculate",
				Description: "Perform a mathematical calculation",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"expression": map[string]interface{}{
							"type":        "string",
							"description": "Mathematical expression to evaluate",
						},
					},
					"required": []string{"expression"},
				},
			},
		},
	}
}

// executeToolCall simulates executing a tool call
func (t *LLMToolCallTester) executeToolCall(toolName string, arguments map[string]interface{}) ToolCallResult {
	switch toolName {
	case "write_file":
		path, pathOk := arguments["path"].(string)
		content, contentOk := arguments["content"].(string)
		if !pathOk || !contentOk || path == "" || content == "" {
			return ToolCallResult{
				ToolName:  toolName,
				Arguments: arguments,
				Success:   false,
				Error:     "Missing path or content",
			}
		}
		return ToolCallResult{
			ToolName:  toolName,
			Arguments: arguments,
			Success:   true,
		}

	case "read_file":
		path, pathOk := arguments["path"].(string)
		if !pathOk || path == "" {
			return ToolCallResult{
				ToolName:  toolName,
				Arguments: arguments,
				Success:   false,
				Error:     "Missing path",
			}
		}
		return ToolCallResult{
			ToolName:  toolName,
			Arguments: arguments,
			Success:   true,
		}

	case "run_command":
		command, commandOk := arguments["command"].(string)
		if !commandOk || command == "" {
			return ToolCallResult{
				ToolName:  toolName,
				Arguments: arguments,
				Success:   false,
				Error:     "Missing command",
			}
		}
		return ToolCallResult{
			ToolName:  toolName,
			Arguments: arguments,
			Success:   true,
		}

	case "calculate":
		expression, expressionOk := arguments["expression"].(string)
		if !expressionOk || expression == "" {
			return ToolCallResult{
				ToolName:  toolName,
				Arguments: arguments,
				Success:   false,
				Error:     "Missing expression",
			}
		}
		// Simple validation - in a real implementation, you'd evaluate the expression
		if strings.Contains(expression, "not_a_number") {
			return ToolCallResult{
				ToolName:  toolName,
				Arguments: arguments,
				Success:   false,
				Error:     "Invalid expression",
			}
		}
		return ToolCallResult{
			ToolName:  toolName,
			Arguments: arguments,
			Success:   true,
		}

	default:
		return ToolCallResult{
			ToolName:  toolName,
			Arguments: arguments,
			Success:   false,
			Error:     fmt.Sprintf("Unknown tool: %s", toolName),
		}
	}
}

// parseToolCallsFromContent parses tool calls from response content
func (t *LLMToolCallTester) parseToolCallsFromContent(content string) []struct {
	Name      string
	Arguments map[string]interface{}
} {
	var toolCalls []struct {
		Name      string
		Arguments map[string]interface{}
	}

	// Parse JSON code blocks
	lines := strings.Split(content, "\n")
	var jsonLines []string
	inCodeBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "```json" {
			inCodeBlock = true
			jsonLines = []string{}
			continue
		}

		if line == "```" && inCodeBlock {
			inCodeBlock = false
			jsonStr := strings.Join(jsonLines, "\n")

			var parsed struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}

			if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
				if parsed.Name != "" {
					toolCalls = append(toolCalls, struct {
						Name      string
						Arguments map[string]interface{}
					}{
						Name:      parsed.Name,
						Arguments: parsed.Arguments,
					})
				}
			}
			continue
		}

		if inCodeBlock {
			jsonLines = append(jsonLines, line)
		}
	}

	// Fallback: parse inline JSON
	if len(toolCalls) == 0 && strings.Contains(content, `"name":`) && strings.Contains(content, `"arguments":`) {
		var parsed struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}

		if err := json.Unmarshal([]byte(content), &parsed); err == nil && parsed.Name != "" {
			toolCalls = append(toolCalls, struct {
				Name      string
				Arguments map[string]interface{}
			}{
				Name:      parsed.Name,
				Arguments: parsed.Arguments,
			})
		}
	}

	return toolCalls
}

// sendChatRequest sends a chat request to the Ollama API
func (t *LLMToolCallTester) sendChatRequest(messages []Message) (*ChatResponse, error) {
	requestData := ChatRequest{
		Model:    t.Model,
		Messages: messages,
		Tools:    t.Tools,
		Stream:   false,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	client := &http.Client{Timeout: 3600 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("%s/api/chat", t.OllamaURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &chatResp, nil
}

// runTest executes a single test case
func (t *LLMToolCallTester) runTest(testCase TestCase) TestResult {
	fmt.Printf("\n🧪 Running test: %s\n", testCase.Name)
	fmt.Printf("   Description: %s\n", testCase.Description)

	startTime := time.Now()

	messages := []Message{
		{Role: "system", Content: testCase.SystemPrompt},
		{Role: "user", Content: testCase.UserMessage},
	}

	var toolCalls []ToolCallResult
	maxIterations := 10

	for iteration := 0; iteration < maxIterations; iteration++ {
		response, err := t.sendChatRequest(messages)
		if err != nil {
			duration := time.Since(startTime).Seconds()
			return TestResult{
				TestName:        testCase.Name,
				Result:          TestStatusFail,
				ToolCalls:       toolCalls,
				ResponseContent: "",
				Duration:        duration,
				Notes:           fmt.Sprintf("Failed to get response from API: %v", err),
			}
		}

		content := response.Message.Content
		apiToolCalls := response.Message.ToolCalls

		// Add assistant message to conversation
		messages = append(messages, Message{
			Role:    "assistant",
			Content: content,
		})

		// Handle API-level tool calls
		if len(apiToolCalls) > 0 {
			for _, toolCall := range apiToolCalls {
				toolName := toolCall.Function.Name
				var arguments map[string]interface{}

				if err := json.Unmarshal(toolCall.Function.Arguments, &arguments); err != nil {
					arguments = make(map[string]interface{})
				}

				result := t.executeToolCall(toolName, arguments)
				toolCalls = append(toolCalls, result)

				// Add tool result to conversation
				toolResult := "Tool executed successfully"
				if !result.Success {
					toolResult = fmt.Sprintf("Tool failed: %s", result.Error)
				}
				messages = append(messages, Message{
					Role:    "tool",
					Content: fmt.Sprintf("Tool %s %s", toolName, toolResult),
				})
			}
		} else if content != "" {
			// Handle content-embedded tool calls
			parsedCalls := t.parseToolCallsFromContent(content)
			if len(parsedCalls) > 0 {
				for _, call := range parsedCalls {
					result := t.executeToolCall(call.Name, call.Arguments)
					toolCalls = append(toolCalls, result)

					// Add tool result to conversation
					toolResult := "executed successfully"
					if !result.Success {
						toolResult = fmt.Sprintf("failed: %s", result.Error)
					}
					messages = append(messages, Message{
						Role:    "tool",
						Content: fmt.Sprintf("Tool %s %s", call.Name, toolResult),
					})
				}
			} else {
				// No tool calls found, conversation complete
				break
			}
		} else {
			// No content or tool calls, conversation complete
			break
		}
	}

	duration := time.Since(startTime).Seconds()
	result := t.evaluateTestResult(testCase, toolCalls, messages[len(messages)-1].Content)

	return TestResult{
		TestName:        testCase.Name,
		Result:          result,
		ToolCalls:       toolCalls,
		ResponseContent: messages[len(messages)-1].Content,
		Duration:        duration,
	}
}

// evaluateTestResult evaluates whether the test passed
func (t *LLMToolCallTester) evaluateTestResult(testCase TestCase, toolCalls []ToolCallResult, content string) TestStatus {
	calledTools := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		calledTools[i] = tc.ToolName
	}

	expectedTools := testCase.ExpectedTools

	if len(expectedTools) == 0 {
		// No specific tools expected, just check if any were called
		if len(toolCalls) > 0 {
			return TestStatusPass
		}
		return TestStatusFail
	}

	// Check if all expected tools were called
	missingTools := []string{}
	for _, expectedTool := range expectedTools {
		found := false
		for _, calledTool := range calledTools {
			if calledTool == expectedTool {
				found = true
				break
			}
		}
		if !found {
			missingTools = append(missingTools, expectedTool)
		}
	}

	if len(missingTools) > 0 {
		if len(toolCalls) > 0 {
			return TestStatusPartial
		}
		return TestStatusFail
	}

	// Check if tool calls were successful
	for _, tc := range toolCalls {
		if !tc.Success {
			return TestStatusPartial
		}
	}

	return TestStatusPass
}

// getTestCases returns the test cases
func (t *LLMToolCallTester) getTestCases() []TestCase {
	return []TestCase{
		{
			Name:            "basic_tool_call",
			Description:     "Test basic tool call recognition and execution",
			SystemPrompt:    "You are a helpful assistant that can use tools to complete tasks.",
			UserMessage:     "Calculate 2 + 2",
			ExpectedTools:   []string{"calculate"},
			SuccessCriteria: "Should call calculate tool with expression '2 + 2'",
			Timeout:         3600,
		},
		{
			Name:            "sequential_tool_calls",
			Description:     "Test multiple tool calls in sequence",
			SystemPrompt:    "You are a helpful assistant that can use tools to complete tasks.",
			UserMessage:     "Write 'Hello World' to a file called hello.txt, then read it back",
			ExpectedTools:   []string{"write_file", "read_file"},
			SuccessCriteria: "Should call write_file then read_file",
			Timeout:         3600,
		},
		{
			Name:            "complex_workflow",
			Description:     "Test complex multi-step workflow with conditional logic",
			SystemPrompt:    "You are a helpful assistant that can use tools to complete tasks.",
			UserMessage:     "Create a Python script that prints 'Hello World', save it as hello.py, then run it",
			ExpectedTools:   []string{"write_file", "run_command"},
			SuccessCriteria: "Should write Python file and execute it",
			Timeout:         3600,
		},
		{
			Name:            "error_handling",
			Description:     "Test how the LLM handles tool call errors",
			SystemPrompt:    "You are a helpful assistant that can use tools to complete tasks.",
			UserMessage:     "Read a file that doesn't exist: /nonexistent/file.txt",
			ExpectedTools:   []string{"read_file"},
			SuccessCriteria: "Should attempt to read the file and handle the error gracefully",
			Timeout:         3600,
		},
		{
			Name:            "parameter_validation",
			Description:     "Test tool call parameter validation",
			SystemPrompt:    "You are a helpful assistant that can use tools to complete tasks.",
			UserMessage:     "Calculate the result of an invalid expression: 'not_a_number + 5'",
			ExpectedTools:   []string{"calculate"},
			SuccessCriteria: "Should call calculate and handle invalid expression",
			Timeout:         3600,
		},
		{
			Name:            "no_tools_needed",
			Description:     "Test response when no tools are needed",
			SystemPrompt:    "You are a helpful assistant that can use tools to complete tasks.",
			UserMessage:     "What is the capital of France?",
			ExpectedTools:   []string{},
			SuccessCriteria: "Should respond directly without using tools",
			Timeout:         3600,
		},
	}
}

// runAllTests executes all test cases
func (t *LLMToolCallTester) runAllTests() map[string]TestResult {
	testCases := t.getTestCases()
	results := make(map[string]TestResult)

	fmt.Printf("🚀 Starting LLM Tool Call Tests for model: %s\n", t.Model)
	fmt.Printf("📍 Ollama URL: %s\n", t.OllamaURL)
	fmt.Printf("📊 Running %d test cases\n", len(testCases))

	for _, testCase := range testCases {
		result := t.runTest(testCase)
		results[testCase.Name] = result
	}

	return results
}

// printSummary prints a summary of test results
func (t *LLMToolCallTester) printSummary(results map[string]TestResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	totalTests := len(results)
	passed := 0
	failed := 0
	partial := 0

	for _, result := range results {
		switch result.Result {
		case TestStatusPass:
			passed++
		case TestStatusFail:
			failed++
		case TestStatusPartial:
			partial++
		}
	}

	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Printf("✅ Passed: %d\n", passed)
	fmt.Printf("❌ Failed: %d\n", failed)
	fmt.Printf("⚠️  Partial: %d\n", partial)
	fmt.Printf("📊 Success Rate: %.1f%%\n", (float64(passed)/float64(totalTests))*100)

	fmt.Println("\n📝 DETAILED RESULTS:")
	for testName, result := range results {
		var statusEmoji string
		switch result.Result {
		case TestStatusPass:
			statusEmoji = "✅"
		case TestStatusFail:
			statusEmoji = "❌"
		case TestStatusPartial:
			statusEmoji = "⚠️"
		default:
			statusEmoji = "❓"
		}

		fmt.Printf("\n%s %s (%s)\n", statusEmoji, testName, result.Result)
		fmt.Printf("   Duration: %.2fs\n", result.Duration)
		fmt.Printf("   Tool Calls: %d\n", len(result.ToolCalls))

		for _, tc := range result.ToolCalls {
			status := "✓"
			if !tc.Success {
				status = "✗"
			}
			fmt.Printf("     %s %s(%v)\n", status, tc.ToolName, tc.Arguments)
		}

		if result.Notes != "" {
			fmt.Printf("   Notes: %s\n", result.Notes)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))

	// Overall assessment
	if passed == totalTests {
		fmt.Println("🎉 EXCELLENT: This LLM has robust tool call support!")
	} else if passed >= int(float64(totalTests)*0.8) {
		fmt.Println("👍 GOOD: This LLM has solid tool call support with minor issues.")
	} else if passed >= int(float64(totalTests)*0.5) {
		fmt.Println("⚠️  MODERATE: This LLM has partial tool call support.")
	} else {
		fmt.Println("❌ POOR: This LLM has limited or broken tool call support.")
	}
}

func main() {
	var (
		ollamaURL = flag.String("ollama-url", "http://localhost:11434", "Ollama server URL")
		model     = flag.String("model", "", "Model name to test (required)")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *model == "" {
		fmt.Println("Error: --model is required")
		flag.Usage()
		os.Exit(1)
	}

	_ = verbose // For future use

	tester := NewLLMToolCallTester(*ollamaURL, *model)

	results := tester.runAllTests()
	tester.printSummary(results)

	// Exit with appropriate code
	totalTests := len(results)
	passed := 0
	for _, result := range results {
		if result.Result == TestStatusPass {
			passed++
		}
	}

	if passed == totalTests {
		os.Exit(0) // All tests passed
	} else if passed >= int(float64(totalTests)*0.8) {
		os.Exit(1) // Most tests passed
	} else {
		os.Exit(2) // Many tests failed
	}
}