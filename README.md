# Wex - Agent Coding System

A containerized LLM-powered coding assistant that provides an isolated environment for code generation and modification.

## Architecture

Wex consists of two main components:

- **Go Engine** (`main.go`): Runs inside Docker container, communicates with Ollama LLM server
- **Python Runner** (`run_engine.py`): Manages Docker container lifecycle and workspace mounting

## Features

- **Containerized Execution**: Isolated environment for safe code operations
- **Ollama Integration**: Connects to local Ollama server for LLM inference
- **Automatic Model Selection**: Uses first available model from Ollama server
- **Workspace Mounting**: Mounts your project directory for file operations
- **Tool System**: Provides LLM with `read_file`, `write_file`, and `run_command` tools
- **Smart Rebuilding**: Automatically rebuilds Docker image when source files change

## Prerequisites

- Docker installed and running
- Ollama server running (default: `http://192.168.0.63:11434`)
- Python 3.7+ for the runner script

## Quick Start

1. **Clone and build:**
   ```bash
   git clone <repository>
   cd wex
   python run_engine.py --build
   ```

2. **Run on current directory:**
   ```bash
   python run_engine.py "Create a hello world program in Python"
   ```

3. **Run on specific project:**
   ```bash
   python run_engine.py --workspace /path/to/project "Add unit tests"
   ```

## Usage

### Basic Commands

```bash
# Send message to coding assistant
python run_engine.py "Create a REST API with Flask"

# Read message from file
python run_engine.py --file prompt.txt

# Work on specific project
python run_engine.py --workspace /path/to/project "Refactor database layer"

# Interactive shell for debugging
python run_engine.py --shell

# Just build the Docker image
python run_engine.py --build
```

### Options

- `--workspace, -w`: Path to workspace directory (default: current directory)
- `--file, -f`: Read message from file instead of command line
- `--ollama-url`: Ollama server URL (default: `http://192.168.0.63:11434`)
- `--ollama-model, -m`: Specific model to use (default: auto-select first available)
- `--build`: Build Docker image and exit
- `--shell`: Start interactive shell in container

## Configuration

### Environment Variables

- `OLLAMA_URL`: Ollama server URL
- `OLLAMA_MODEL`: Specific model name (optional)
- `WORKSPACE`: Workspace directory inside container

### System Prompt

The LLM behavior is configured via `system_prompt.txt`. This file contains instructions that are sent to the LLM at the start of each conversation.

## How It Works

1. **Python Runner** checks if Docker image needs rebuilding based on file timestamps
2. **Container Launch** with workspace mounted as `/workspace` volume
3. **Go Engine** starts and loads system prompt from `system_prompt.txt`
4. **LLM Communication** via Ollama API with tool call support
5. **Tool Execution** for file operations and command execution within workspace

## File Structure

```
wex/
├── main.go              # Go engine (runs in container)
├── system_prompt.txt    # LLM instructions
├── Dockerfile          # Container configuration
├── run_engine.py       # Python runner script
├── go.mod              # Go dependencies
└── README.md           # This file
```

## Development

### Building Manually

```bash
# Build Docker image
docker build -t wex:latest .

# Run container with workspace
docker run --rm -v $(pwd):/workspace wex:latest "your message"
```

### Debugging

```bash
# Interactive shell in container
python run_engine.py --shell

# Check container logs
docker logs wex-engine
```

## Architecture Details

### Workspace Isolation

The workspace directory is mounted as a volume to provide:
- **Isolation**: Engine code separate from target code
- **Security**: Container only accesses specified workspace
- **Flexibility**: Work on any project by changing workspace path

### Tool System

The engine provides three tools to the LLM:
- `read_file(path)`: Read file contents from workspace
- `write_file(path, content)`: Write content to file in workspace
- `run_command(command, timeout)`: Execute shell command in workspace

### Auto-Rebuild

The runner automatically rebuilds the Docker image when any of these files change:
- `main.go`
- `go.mod`
- `go.sum`
- `Dockerfile`
- `system_prompt.txt`

This ensures the engine stays up-to-date with code changes during development.

## License

[License information here]