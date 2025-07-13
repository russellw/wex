#!/usr/bin/env python3
"""
Docker runner for the wex agent coding engine.
Manages Docker container lifecycle and workspace mounting.
"""

import argparse
import os
import sys
import subprocess
import tempfile
import shutil
from pathlib import Path


class WexEngine:
    def __init__(self, workspace_path=None, ollama_url=None, ollama_model=None):
        self.workspace_path = workspace_path or os.getcwd()
        self.ollama_url = ollama_url or "http://192.168.0.63:11434"
        self.ollama_model = ollama_model or ""
        self.container_name = "wex-engine"
        self.image_name = "wex:latest"
        
    def build_image(self):
        """Build the Docker image for the engine."""
        print("Building Docker image...")
        try:
            result = subprocess.run([
                "docker", "build", "-t", self.image_name, "."
            ], cwd=Path(__file__).parent, check=True, capture_output=True, text=True)
            print("Docker image built successfully")
            return True
        except subprocess.CalledProcessError as e:
            print(f"Failed to build Docker image: {e}")
            print(f"stdout: {e.stdout}")
            print(f"stderr: {e.stderr}")
            return False
    
    def check_image_exists(self):
        """Check if the Docker image exists."""
        try:
            result = subprocess.run([
                "docker", "images", "-q", self.image_name
            ], capture_output=True, text=True, check=True)
            return bool(result.stdout.strip())
        except subprocess.CalledProcessError:
            return False
    
    def stop_existing_container(self):
        """Stop and remove any existing container with the same name."""
        try:
            subprocess.run([
                "docker", "stop", self.container_name
            ], capture_output=True, check=True)
            print(f"Stopped existing container: {self.container_name}")
        except subprocess.CalledProcessError:
            pass  # Container wasn't running
        
        try:
            subprocess.run([
                "docker", "rm", self.container_name
            ], capture_output=True, check=True)
            print(f"Removed existing container: {self.container_name}")
        except subprocess.CalledProcessError:
            pass  # Container didn't exist
    
    def run_engine(self, message):
        """Run the engine in a Docker container with the given message."""
        # Ensure workspace path is absolute
        workspace_path = os.path.abspath(self.workspace_path)
        
        # Check if image exists, build if not
        if not self.check_image_exists():
            print(f"Docker image {self.image_name} not found")
            if not self.build_image():
                return False
        
        # Stop any existing container
        self.stop_existing_container()
        
        # Prepare Docker command
        docker_cmd = [
            "docker", "run",
            "--name", self.container_name,
            "--rm",
            "-v", f"{workspace_path}:/workspace",
            "-e", f"OLLAMA_URL={self.ollama_url}",
            "-e", f"WORKSPACE=/workspace"
        ]
        
        # Add model environment variable if specified
        if self.ollama_model:
            docker_cmd.extend(["-e", f"OLLAMA_MODEL={self.ollama_model}"])
        
        # Add image and message
        docker_cmd.extend([self.image_name, message])
        
        print(f"Running engine with workspace: {workspace_path}")
        print(f"Ollama URL: {self.ollama_url}")
        if self.ollama_model:
            print(f"Model: {self.ollama_model}")
        else:
            print("Model: Auto-select from server")
        print()
        
        try:
            # Run the container interactively
            result = subprocess.run(docker_cmd, check=True)
            return True
        except subprocess.CalledProcessError as e:
            print(f"Engine execution failed: {e}")
            return False
        except KeyboardInterrupt:
            print("\nInterrupted by user")
            # Stop the container if it's still running
            try:
                subprocess.run(["docker", "stop", self.container_name], 
                             capture_output=True, check=True)
            except subprocess.CalledProcessError:
                pass
            return False
    
    def shell(self):
        """Start an interactive shell in the container."""
        # Ensure workspace path is absolute
        workspace_path = os.path.abspath(self.workspace_path)
        
        # Check if image exists, build if not
        if not self.check_image_exists():
            print(f"Docker image {self.image_name} not found")
            if not self.build_image():
                return False
        
        # Stop any existing container
        self.stop_existing_container()
        
        # Prepare Docker command for interactive shell
        docker_cmd = [
            "docker", "run",
            "--name", self.container_name,
            "--rm", "-it",
            "-v", f"{workspace_path}:/workspace",
            "-e", f"OLLAMA_URL={self.ollama_url}",
            "-e", f"WORKSPACE=/workspace",
            "--entrypoint", "/bin/sh",
            self.image_name
        ]
        
        print(f"Starting interactive shell with workspace: {workspace_path}")
        print("Type 'exit' to leave the container")
        print()
        
        try:
            subprocess.run(docker_cmd, check=True)
            return True
        except subprocess.CalledProcessError as e:
            print(f"Shell execution failed: {e}")
            return False
        except KeyboardInterrupt:
            print("\nExiting shell")
            return True


def main():
    parser = argparse.ArgumentParser(
        description="Run the wex agent coding engine in Docker",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python run_engine.py "Create a hello world program in Python"
  python run_engine.py --workspace /path/to/project "Add unit tests"
  python run_engine.py --shell  # Interactive shell
  python run_engine.py --build  # Just build the image
        """
    )
    
    parser.add_argument("message", nargs="?", 
                       help="Message to send to the coding assistant")
    parser.add_argument("--workspace", "-w", 
                       help="Path to workspace directory (default: current directory)")
    parser.add_argument("--ollama-url", 
                       help="Ollama server URL (default: http://192.168.0.63:11434)")
    parser.add_argument("--ollama-model", "-m", 
                       help="Ollama model to use (default: auto-select)")
    parser.add_argument("--build", action="store_true",
                       help="Build Docker image and exit")
    parser.add_argument("--shell", action="store_true",
                       help="Start interactive shell in container")
    
    args = parser.parse_args()
    
    # Create engine instance
    engine = WexEngine(
        workspace_path=args.workspace,
        ollama_url=args.ollama_url,
        ollama_model=args.ollama_model
    )
    
    # Handle different modes
    if args.build:
        success = engine.build_image()
        sys.exit(0 if success else 1)
    
    if args.shell:
        success = engine.shell()
        sys.exit(0 if success else 1)
    
    if not args.message:
        parser.error("Message is required unless using --build or --shell")
    
    # Run the engine with the message
    success = engine.run_engine(args.message)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()