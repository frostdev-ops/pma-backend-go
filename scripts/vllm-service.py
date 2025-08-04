#!/usr/bin/env python3
"""
vLLM Service Manager for LiquidAI LFM2-1.2B
Manages the vLLM server with proper configuration for LFM2 model
Based on specifications from https://huggingface.co/LiquidAI/LFM2-1.2B
"""

import subprocess
import sys
import os
import signal
import json
import time
import requests
import logging
from pathlib import Path
from typing import Optional, Dict, Any

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/var/log/vllm-service.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger('vllm-service')

class VLLMService:
    """Manages vLLM server for LFM2-1.2B model"""
    
    def __init__(self):
        self.venv_path = "/opt/pma/backend/.venv"
        self.python_path = f"{self.venv_path}/bin/python"
        self.model_id = "LiquidAI/LFM2-1.2B-GGUF"
        self.gguf_file = "LFM2-1.2B-Q4_0.gguf"
        self.host = "127.0.0.1"
        self.port = 8000
        self.api_key = None  # Optional for local deployment
        self.process: Optional[subprocess.Popen] = None
        self.config = self._load_config()
        
    def _load_config(self) -> Dict[str, Any]:
        """Load vLLM configuration"""
        return {
            # LFM2 GGUF parameters
            "model": self.model_id,
            "load-format": "gguf",
            "host": self.host,
            "port": self.port,
            "served-model-name": "LFM2-1.2B",
            
            # Performance optimizations for GGUF Q4_0
            "dtype": "auto",  # Auto-detect for GGUF
            "max-model-len": 8192,  # Reduced for better performance
            "max-num-batched-tokens": 8192,
            "max-num-seqs": 16,
            
            # Generation parameters from HuggingFace recommendations
            "temperature": 0.3,
            "min-p": 0.15,
            "repetition-penalty": 1.05,
            
            # Tool use support
            "enable-chat-template": True,
            "chat-template": self._get_lfm2_chat_template(),
            
            # OpenAI compatibility
            "api-key": self.api_key,
            "disable-log-stats": False,
            "disable-log-requests": False,
            
            # Hardware optimization
            "tensor-parallel-size": 1,  # Single GPU/CPU
            "pipeline-parallel-size": 1,
            "worker-use-ray": False,  # Disable Ray for simpler deployment
            
            # Memory management (optimized for GGUF)
            "gpu-memory-utilization": 0.6,  # Reduced for RPi5
            "swap-space": 2,  # Reduced swap usage
            "enforce-eager": False,
        }
    
    def _get_lfm2_chat_template(self) -> str:
        """Return LFM2 ChatML-like template from HuggingFace page"""
        return (
            "{% if messages[0]['role'] == 'system' %}"
            "{% set loop_messages = messages[1:] %}"
            "{% set system_message = messages[0]['content'] %}"
            "{% else %}"
            "{% set loop_messages = messages %}"
            "{% set system_message = 'You are a helpful assistant trained by Liquid AI.' %}"
            "{% endif %}"
            "<|startoftext|><|im_start|>system\n"
            "{{ system_message }}<|im_end|>\n"
            "{% for message in loop_messages %}"
            "<|im_start|>{{ message['role'] }}\n"
            "{{ message['content'] }}<|im_end|>\n"
            "{% endfor %}"
            "{% if add_generation_prompt %}"
            "<|im_start|>assistant\n"
            "{% endif %}"
        )
    
    def _build_command(self) -> list:
        """Build vLLM server command with all parameters"""
        cmd = [
            self.python_path, "-m", "vllm.entrypoints.openai.api_server",
            "--model", self.config["model"],
            "--load-format", self.config["load-format"],
            "--host", self.config["host"],
            "--port", str(self.config["port"]),
            "--served-model-name", self.config["served-model-name"],
            "--dtype", self.config["dtype"],
            "--max-model-len", str(self.config["max-model-len"]),
            "--max-num-batched-tokens", str(self.config["max-num-batched-tokens"]),
            "--max-num-seqs", str(self.config["max-num-seqs"]),
            "--tensor-parallel-size", str(self.config["tensor-parallel-size"]),
            "--pipeline-parallel-size", str(self.config["pipeline-parallel-size"]),
            "--gpu-memory-utilization", str(self.config["gpu-memory-utilization"]),
            "--swap-space", str(self.config["swap-space"]),
        ]
        
        # Add optional flags
        if self.config.get("enable-chat-template"):
            cmd.extend(["--enable-chat-template"])
        
        if self.config.get("disable-log-stats"):
            cmd.extend(["--disable-log-stats"])
        
        if self.config.get("disable-log-requests"):
            cmd.extend(["--disable-log-requests"])
        
        if self.config.get("worker-use-ray") is False:
            cmd.extend(["--worker-use-ray", "False"])
        
        if self.config.get("enforce-eager"):
            cmd.extend(["--enforce-eager"])
        
        if self.config.get("api-key"):
            cmd.extend(["--api-key", self.config["api-key"]])
        
        return cmd
    
    def start(self) -> bool:
        """Start the vLLM server"""
        if self.is_running():
            logger.info("vLLM server is already running")
            return True
        
        logger.info(f"Starting vLLM server for {self.model_id}")
        logger.info(f"Using Python: {self.python_path}")
        logger.info(f"Server will be available at http://{self.host}:{self.port}")
        
        # Check if venv exists
        if not os.path.exists(self.python_path):
            logger.error(f"Python executable not found at {self.python_path}")
            return False
        
        try:
            cmd = self._build_command()
            logger.info(f"Executing command: {' '.join(cmd)}")
            
            # Start the process
            self.process = subprocess.Popen(
                cmd,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                universal_newlines=True,
                preexec_fn=os.setsid  # Create new process group
            )
            
            # Wait for server to be ready
            if self._wait_for_server():
                logger.info("vLLM server started successfully")
                return True
            else:
                logger.error("Failed to start vLLM server")
                self.stop()
                return False
                
        except Exception as e:
            logger.error(f"Failed to start vLLM server: {e}")
            return False
    
    def stop(self) -> bool:
        """Stop the vLLM server"""
        if not self.process:
            logger.info("No vLLM process to stop")
            return True
        
        try:
            logger.info("Stopping vLLM server")
            
            # Send SIGTERM to the process group
            os.killpg(os.getpgid(self.process.pid), signal.SIGTERM)
            
            # Wait for graceful shutdown
            try:
                self.process.wait(timeout=30)
                logger.info("vLLM server stopped gracefully")
            except subprocess.TimeoutExpired:
                logger.warning("Graceful shutdown timed out, forcing kill")
                os.killpg(os.getpgid(self.process.pid), signal.SIGKILL)
                self.process.wait()
            
            self.process = None
            return True
            
        except Exception as e:
            logger.error(f"Failed to stop vLLM server: {e}")
            return False
    
    def restart(self) -> bool:
        """Restart the vLLM server"""
        logger.info("Restarting vLLM server")
        self.stop()
        time.sleep(2)
        return self.start()
    
    def is_running(self) -> bool:
        """Check if vLLM server is running"""
        if self.process and self.process.poll() is None:
            return True
        
        # Check if server responds to health check
        try:
            response = requests.get(
                f"http://{self.host}:{self.port}/health",
                timeout=5
            )
            return response.status_code == 200
        except:
            return False
    
    def _wait_for_server(self, max_wait: int = 120) -> bool:
        """Wait for server to be ready"""
        logger.info("Waiting for vLLM server to be ready...")
        
        for i in range(max_wait):
            if self.process and self.process.poll() is not None:
                logger.error("vLLM process exited unexpectedly")
                return False
            
            try:
                response = requests.get(
                    f"http://{self.host}:{self.port}/health",
                    timeout=2
                )
                if response.status_code == 200:
                    logger.info(f"vLLM server ready after {i+1} seconds")
                    return True
            except requests.RequestException:
                pass
            
            time.sleep(1)
            if i % 10 == 0:
                logger.info(f"Still waiting... ({i}/{max_wait})")
        
        logger.error(f"Server failed to become ready within {max_wait} seconds")
        return False
    
    def get_status(self) -> Dict[str, Any]:
        """Get detailed server status"""
        status = {
            "running": self.is_running(),
            "model": self.model_id,
            "host": self.host,
            "port": self.port,
            "process_id": self.process.pid if self.process else None,
        }
        
        if status["running"]:
            try:
                # Get model info
                response = requests.get(
                    f"http://{self.host}:{self.port}/v1/models",
                    timeout=5
                )
                if response.status_code == 200:
                    status["models"] = response.json()
                
                # Get server stats if available
                try:
                    stats_response = requests.get(
                        f"http://{self.host}:{self.port}/stats",
                        timeout=5
                    )
                    if stats_response.status_code == 200:
                        status["stats"] = stats_response.json()
                except:
                    pass
                    
            except Exception as e:
                status["error"] = str(e)
        
        return status
    
    def test_model(self) -> Dict[str, Any]:
        """Test the model with a simple request"""
        if not self.is_running():
            return {"error": "Server is not running"}
        
        try:
            # Test with a simple chat completion
            response = requests.post(
                f"http://{self.host}:{self.port}/v1/chat/completions",
                json={
                    "model": "LFM2-1.2B",
                    "messages": [
                        {"role": "user", "content": "What is C. elegans?"}
                    ],
                    "temperature": 0.3,
                    "max_tokens": 100
                },
                headers={"Content-Type": "application/json"},
                timeout=30
            )
            
            if response.status_code == 200:
                result = response.json()
                return {
                    "success": True,
                    "response": result["choices"][0]["message"]["content"],
                    "usage": result.get("usage", {}),
                    "model": result.get("model")
                }
            else:
                return {
                    "error": f"HTTP {response.status_code}: {response.text}"
                }
                
        except Exception as e:
            return {"error": str(e)}

def main():
    """Main function for command-line usage"""
    if len(sys.argv) < 2:
        print("Usage: python vllm-service.py {start|stop|restart|status|test}")
        sys.exit(1)
    
    service = VLLMService()
    command = sys.argv[1].lower()
    
    if command == "start":
        success = service.start()
        sys.exit(0 if success else 1)
    
    elif command == "stop":
        success = service.stop()
        sys.exit(0 if success else 1)
    
    elif command == "restart":
        success = service.restart()
        sys.exit(0 if success else 1)
    
    elif command == "status":
        status = service.get_status()
        print(json.dumps(status, indent=2))
        sys.exit(0 if status["running"] else 1)
    
    elif command == "test":
        result = service.test_model()
        print(json.dumps(result, indent=2))
        sys.exit(0 if result.get("success") else 1)
    
    else:
        print(f"Unknown command: {command}")
        print("Usage: python vllm-service.py {start|stop|restart|status|test}")
        sys.exit(1)

if __name__ == "__main__":
    main()