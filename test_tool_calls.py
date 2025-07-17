#!/usr/bin/env python3
"""
Comprehensive test script to evaluate whether an LLM genuinely supports tool calls.

This script tests various aspects of tool call support:
1. Basic tool call execution
2. Tool call format recognition (JSON vs markdown)
3. Sequential tool execution
4. Error handling and recovery
5. Complex multi-step workflows
6. Tool call parameter validation

Usage: python test_tool_calls.py --ollama-url http://localhost:11434 --model model-name
"""

import argparse
import json
import sys
import time
from typing import List, Dict, Any, Optional, Tuple
import requests
from dataclasses import dataclass
from enum import Enum


class TestStatus(Enum):
    PASS = "PASS"
    FAIL = "FAIL"
    PARTIAL = "PARTIAL"
    SKIP = "SKIP"


@dataclass
class TestCase:
    name: str
    description: str
    system_prompt: str
    user_message: str
    expected_tools: List[str]
    success_criteria: str
    timeout: int = 3600


@dataclass
class ToolCallResult:
    tool_name: str
    arguments: Dict[str, Any]
    success: bool
    error: Optional[str] = None


@dataclass
class TestResult:
    test_name: str
    result: TestStatus
    tool_calls: List[ToolCallResult]
    response_content: str
    duration: float
    notes: str = ""


class LLMToolCallTester:
    def __init__(self, ollama_url: str, model: str):
        self.ollama_url = ollama_url.rstrip('/')
        self.model = model
        self.tools = self._get_test_tools()
        
    def _get_test_tools(self) -> List[Dict[str, Any]]:
        """Define the test tools available to the LLM"""
        return [
            {
                "type": "function",
                "function": {
                    "name": "write_file",
                    "description": "Write content to a file",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "path": {
                                "type": "string",
                                "description": "Path to the file to write"
                            },
                            "content": {
                                "type": "string",
                                "description": "Content to write to the file"
                            }
                        },
                        "required": ["path", "content"]
                    }
                }
            },
            {
                "type": "function",
                "function": {
                    "name": "read_file",
                    "description": "Read the contents of a file",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "path": {
                                "type": "string",
                                "description": "Path to the file to read"
                            }
                        },
                        "required": ["path"]
                    }
                }
            },
            {
                "type": "function",
                "function": {
                    "name": "run_command",
                    "description": "Execute a shell command",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "command": {
                                "type": "string",
                                "description": "Shell command to execute"
                            },
                            "timeout": {
                                "type": "number",
                                "description": "Timeout in seconds (optional, default 30)"
                            }
                        },
                        "required": ["command"]
                    }
                }
            },
            {
                "type": "function",
                "function": {
                    "name": "calculate",
                    "description": "Perform a mathematical calculation",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "expression": {
                                "type": "string",
                                "description": "Mathematical expression to evaluate"
                            }
                        },
                        "required": ["expression"]
                    }
                }
            }
        ]
    
    def _execute_tool_call(self, tool_name: str, arguments: Dict[str, Any]) -> ToolCallResult:
        """Mock execute a tool call and return the result"""
        try:
            if tool_name == "write_file":
                # Simulate writing a file
                path = arguments.get("path", "")
                content = arguments.get("content", "")
                if not path or not content:
                    return ToolCallResult(tool_name, arguments, False, "Missing path or content")
                return ToolCallResult(tool_name, arguments, True)
            
            elif tool_name == "read_file":
                # Simulate reading a file
                path = arguments.get("path", "")
                if not path:
                    return ToolCallResult(tool_name, arguments, False, "Missing path")
                return ToolCallResult(tool_name, arguments, True)
            
            elif tool_name == "run_command":
                # Simulate running a command
                command = arguments.get("command", "")
                if not command:
                    return ToolCallResult(tool_name, arguments, False, "Missing command")
                return ToolCallResult(tool_name, arguments, True)
            
            elif tool_name == "calculate":
                # Simulate calculation
                expression = arguments.get("expression", "")
                if not expression:
                    return ToolCallResult(tool_name, arguments, False, "Missing expression")
                try:
                    # Simple eval for testing (unsafe in production)
                    result = eval(expression)
                    return ToolCallResult(tool_name, arguments, True)
                except:
                    return ToolCallResult(tool_name, arguments, False, "Invalid expression")
            
            else:
                return ToolCallResult(tool_name, arguments, False, f"Unknown tool: {tool_name}")
        
        except Exception as e:
            return ToolCallResult(tool_name, arguments, False, str(e))
    
    def _parse_tool_calls_from_response(self, content: str) -> List[Tuple[str, Dict[str, Any]]]:
        """Parse tool calls from LLM response, handling various formats"""
        tool_calls = []
        
        # Method 1: Try to parse standard OpenAI-style tool calls
        # This would be handled by the API response format
        
        # Method 2: Parse JSON code blocks
        lines = content.split('\n')
        json_lines = []
        in_code_block = False
        
        for line in lines:
            line = line.strip()
            
            if line == "```json":
                in_code_block = True
                json_lines = []
                continue
            
            if line == "```" and in_code_block:
                in_code_block = False
                json_str = '\n'.join(json_lines)
                try:
                    parsed = json.loads(json_str)
                    if isinstance(parsed, dict) and "name" in parsed and "arguments" in parsed:
                        tool_calls.append((parsed["name"], parsed["arguments"]))
                except json.JSONDecodeError:
                    pass
                continue
            
            if in_code_block:
                json_lines.append(line)
        
        # Method 3: Parse inline JSON
        if not tool_calls:
            try:
                parsed = json.loads(content)
                if isinstance(parsed, dict) and "name" in parsed and "arguments" in parsed:
                    tool_calls.append((parsed["name"], parsed["arguments"]))
            except json.JSONDecodeError:
                pass
        
        return tool_calls
    
    def _send_chat_request(self, messages: List[Dict[str, str]]) -> Optional[Dict[str, Any]]:
        """Send a chat request to the Ollama API"""
        request_data = {
            "model": self.model,
            "messages": messages,
            "tools": self.tools,
            "stream": False
        }
        
        try:
            response = requests.post(
                f"{self.ollama_url}/api/chat",
                json=request_data,
                headers={"Content-Type": "application/json"},
                timeout=3600
            )
            
            if response.status_code == 200:
                return response.json()
            else:
                print(f"API Error: {response.status_code} - {response.text}")
                return None
                
        except requests.exceptions.RequestException as e:
            print(f"Request failed: {e}")
            return None
    
    def run_test(self, test_case: TestCase) -> TestResult:
        """Run a single test case"""
        print(f"\n🧪 Running test: {test_case.name}")
        print(f"   Description: {test_case.description}")
        
        start_time = time.time()
        
        messages = [
            {"role": "system", "content": test_case.system_prompt},
            {"role": "user", "content": test_case.user_message}
        ]
        
        tool_calls = []
        max_iterations = 10
        iteration = 0
        
        while iteration < max_iterations:
            iteration += 1
            response = self._send_chat_request(messages)
            
            if not response:
                duration = time.time() - start_time
                return TestResult(
                    test_case.name,
                    TestStatus.FAIL,
                    tool_calls,
                    "",
                    duration,
                    "Failed to get response from API"
                )
            
            assistant_message = response.get("message", {})
            content = assistant_message.get("content", "")
            api_tool_calls = assistant_message.get("tool_calls", [])
            
            # Add assistant message to conversation
            messages.append({
                "role": "assistant",
                "content": content
            })
            
            # Handle API-level tool calls
            if api_tool_calls:
                for tool_call in api_tool_calls:
                    tool_name = tool_call.get("function", {}).get("name", "")
                    arguments = tool_call.get("function", {}).get("arguments", {})
                    
                    if isinstance(arguments, str):
                        try:
                            arguments = json.loads(arguments)
                        except json.JSONDecodeError:
                            arguments = {}
                    
                    result = self._execute_tool_call(tool_name, arguments)
                    tool_calls.append(result)
                    
                    # Add tool result to conversation
                    messages.append({
                        "role": "tool",
                        "content": f"Tool {tool_name} executed successfully" if result.success else f"Tool {tool_name} failed: {result.error}"
                    })
            
            # Handle content-embedded tool calls
            elif content:
                parsed_calls = self._parse_tool_calls_from_response(content)
                if parsed_calls:
                    for tool_name, arguments in parsed_calls:
                        result = self._execute_tool_call(tool_name, arguments)
                        tool_calls.append(result)
                        
                        # Add tool result to conversation
                        messages.append({
                            "role": "tool", 
                            "content": f"Tool {tool_name} executed successfully" if result.success else f"Tool {tool_name} failed: {result.error}"
                        })
                else:
                    # No tool calls found, conversation complete
                    break
            else:
                # No content or tool calls, conversation complete
                break
        
        duration = time.time() - start_time
        
        # Evaluate test result
        result = self._evaluate_test_result(test_case, tool_calls, content)
        
        return TestResult(
            test_case.name,
            result,
            tool_calls,
            content,
            duration
        )
    
    def _evaluate_test_result(self, test_case: TestCase, tool_calls: List[ToolCallResult], content: str) -> TestStatus:
        """Evaluate whether the test passed based on the criteria"""
        
        # Check if expected tools were called
        called_tools = [tc.tool_name for tc in tool_calls]
        expected_tools = test_case.expected_tools
        
        if not expected_tools:
            # No specific tools expected, just check if any were called
            return TestStatus.PASS if tool_calls else TestStatus.FAIL
        
        # Check if all expected tools were called
        missing_tools = [tool for tool in expected_tools if tool not in called_tools]
        if missing_tools:
            return TestStatus.PARTIAL if tool_calls else TestStatus.FAIL
        
        # Check if tool calls were successful
        failed_calls = [tc for tc in tool_calls if not tc.success]
        if failed_calls:
            return TestStatus.PARTIAL
        
        return TestStatus.PASS
    
    def get_test_cases(self) -> List[TestCase]:
        """Define the test cases"""
        return [
            TestCase(
                name="basic_tool_call",
                description="Test basic tool call recognition and execution",
                system_prompt="You are a helpful assistant that can use tools to complete tasks.",
                user_message="Calculate 2 + 2",
                expected_tools=["calculate"],
                success_criteria="Should call calculate tool with expression '2 + 2'"
            ),
            
            TestCase(
                name="sequential_tool_calls",
                description="Test multiple tool calls in sequence",
                system_prompt="You are a helpful assistant that can use tools to complete tasks.",
                user_message="Write 'Hello World' to a file called hello.txt, then read it back",
                expected_tools=["write_file", "read_file"],
                success_criteria="Should call write_file then read_file"
            ),
            
            TestCase(
                name="complex_workflow",
                description="Test complex multi-step workflow with conditional logic",
                system_prompt="You are a helpful assistant that can use tools to complete tasks.",
                user_message="Create a Python script that prints 'Hello World', save it as hello.py, then run it",
                expected_tools=["write_file", "run_command"],
                success_criteria="Should write Python file and execute it"
            ),
            
            TestCase(
                name="error_handling",
                description="Test how the LLM handles tool call errors",
                system_prompt="You are a helpful assistant that can use tools to complete tasks.",
                user_message="Read a file that doesn't exist: /nonexistent/file.txt",
                expected_tools=["read_file"],
                success_criteria="Should attempt to read the file and handle the error gracefully"
            ),
            
            TestCase(
                name="parameter_validation",
                description="Test tool call parameter validation",
                system_prompt="You are a helpful assistant that can use tools to complete tasks.",
                user_message="Calculate the result of an invalid expression: 'not_a_number + 5'",
                expected_tools=["calculate"],
                success_criteria="Should call calculate and handle invalid expression"
            ),
            
            TestCase(
                name="no_tools_needed",
                description="Test response when no tools are needed",
                system_prompt="You are a helpful assistant that can use tools to complete tasks.",
                user_message="What is the capital of France?",
                expected_tools=[],
                success_criteria="Should respond directly without using tools"
            )
        ]
    
    def run_all_tests(self) -> Dict[str, TestResult]:
        """Run all test cases and return results"""
        test_cases = self.get_test_cases()
        results = {}
        
        print(f"🚀 Starting LLM Tool Call Tests for model: {self.model}")
        print(f"📍 Ollama URL: {self.ollama_url}")
        print(f"📊 Running {len(test_cases)} test cases")
        
        for test_case in test_cases:
            result = self.run_test(test_case)
            results[test_case.name] = result
        
        return results
    
    def print_summary(self, results: Dict[str, TestResult]):
        """Print a summary of test results"""
        print("\n" + "="*60)
        print("📋 TEST SUMMARY")
        print("="*60)
        
        total_tests = len(results)
        passed = sum(1 for r in results.values() if r.result == TestStatus.PASS)
        failed = sum(1 for r in results.values() if r.result == TestStatus.FAIL)
        partial = sum(1 for r in results.values() if r.result == TestStatus.PARTIAL)
        
        print(f"Total Tests: {total_tests}")
        print(f"✅ Passed: {passed}")
        print(f"❌ Failed: {failed}")
        print(f"⚠️  Partial: {partial}")
        print(f"📊 Success Rate: {(passed/total_tests)*100:.1f}%")
        
        print("\n📝 DETAILED RESULTS:")
        for test_name, result in results.items():
            status_emoji = {
                TestStatus.PASS: "✅",
                TestStatus.FAIL: "❌", 
                TestStatus.PARTIAL: "⚠️"
            }.get(result.result, "❓")
            
            print(f"\n{status_emoji} {test_name} ({result.result.value})")
            print(f"   Duration: {result.duration:.2f}s")
            print(f"   Tool Calls: {len(result.tool_calls)}")
            
            if result.tool_calls:
                for tc in result.tool_calls:
                    status = "✓" if tc.success else "✗"
                    print(f"     {status} {tc.tool_name}({tc.arguments})")
            
            if result.notes:
                print(f"   Notes: {result.notes}")
        
        print("\n" + "="*60)
        
        # Overall assessment
        if passed == total_tests:
            print("🎉 EXCELLENT: This LLM has robust tool call support!")
        elif passed >= total_tests * 0.8:
            print("👍 GOOD: This LLM has solid tool call support with minor issues.")
        elif passed >= total_tests * 0.5:
            print("⚠️  MODERATE: This LLM has partial tool call support.")
        else:
            print("❌ POOR: This LLM has limited or broken tool call support.")


def main():
    parser = argparse.ArgumentParser(description="Test LLM tool call support")
    parser.add_argument("--ollama-url", default="http://localhost:11434", 
                       help="Ollama server URL")
    parser.add_argument("--model", required=True, 
                       help="Model name to test")
    parser.add_argument("--verbose", "-v", action="store_true",
                       help="Enable verbose output")
    
    args = parser.parse_args()
    
    tester = LLMToolCallTester(args.ollama_url, args.model)
    
    try:
        results = tester.run_all_tests()
        tester.print_summary(results)
        
        # Exit with appropriate code
        passed = sum(1 for r in results.values() if r.result == TestStatus.PASS)
        total = len(results)
        
        if passed == total:
            sys.exit(0)  # All tests passed
        elif passed >= total * 0.8:
            sys.exit(1)  # Most tests passed
        else:
            sys.exit(2)  # Many tests failed
            
    except KeyboardInterrupt:
        print("\n⚠️  Tests interrupted by user")
        sys.exit(3)
    except Exception as e:
        print(f"❌ Test execution failed: {e}")
        sys.exit(4)


if __name__ == "__main__":
    main()