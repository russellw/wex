This project is an agent coding system
The system will communicate with an LLM
The LLM will run on an Ollama server at 192.168.0.63
The main engine of the system is a program to be written in Go
The Go program will run within a Docker container, along with the target code that it is working on, and any dependencies, generated files and such needed by the target code
Test scripts, user interface and such, that run outside the container, control the main engine and retrieve target code when it has been written, are to be written in Python
The engine will provide the LLM with tools, using the usual LLM tool call protocol
Among the provided tools will be read_file, write_file and run_command
