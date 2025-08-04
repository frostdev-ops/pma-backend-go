#!/usr/bin/env python3
"""
vLLM Monitoring Script for LFM2-1.2B
Monitors vLLM server health, performance, and provides detailed metrics
"""

import time
import json
import psutil
import requests
import argparse
import signal
import sys
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from dataclasses import dataclass
from pathlib import Path

@dataclass
class VLLMMetrics:
    """Data class for vLLM metrics"""
    timestamp: datetime
    status: str
    response_time: float
    memory_usage: float
    cpu_usage: float
    gpu_memory: Optional[float] = None
    active_requests: int = 0
    completed_requests: int = 0
    error_count: int = 0

class VLLMMonitor:
    """Monitor vLLM server performance and health"""
    
    def __init__(self, host: str = "127.0.0.1", port: int = 8000):
        self.host = host
        self.port = port
        self.base_url = f"http://{host}:{port}"
        self.metrics_history: List[VLLMMetrics] = []
        self.max_history = 1000  # Keep last 1000 metrics
        self.running = True
        
        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        print(f"\nReceived signal {signum}, shutting down...")
        self.running = False
    
    def check_health(self) -> Dict[str, Any]:
        """Check vLLM server health"""
        try:
            start_time = time.time()
            response = requests.get(f"{self.base_url}/health", timeout=10)
            response_time = time.time() - start_time
            
            if response.status_code == 200:
                return {
                    "status": "healthy",
                    "response_time": response_time,
                    "details": response.json() if response.content else {}
                }
            else:
                return {
                    "status": "unhealthy",
                    "response_time": response_time,
                    "error": f"HTTP {response.status_code}"
                }
        except requests.RequestException as e:
            return {
                "status": "error",
                "response_time": 0.0,
                "error": str(e)
            }
    
    def get_models(self) -> Dict[str, Any]:
        """Get available models"""
        try:
            response = requests.get(f"{self.base_url}/v1/models", timeout=10)
            if response.status_code == 200:
                return response.json()
            else:
                return {"error": f"HTTP {response.status_code}"}
        except requests.RequestException as e:
            return {"error": str(e)}
    
    def get_server_stats(self) -> Dict[str, Any]:
        """Get server statistics if available"""
        try:
            response = requests.get(f"{self.base_url}/stats", timeout=5)
            if response.status_code == 200:
                return response.json()
            else:
                return {}
        except:
            return {}
    
    def get_system_metrics(self) -> Dict[str, Any]:
        """Get system metrics (CPU, memory, etc.)"""
        metrics = {
            "cpu_percent": psutil.cpu_percent(interval=1),
            "memory": psutil.virtual_memory()._asdict(),
            "disk": psutil.disk_usage('/')._asdict(),
            "network": {}
        }
        
        # Get network stats
        net_io = psutil.net_io_counters()
        if net_io:
            metrics["network"] = {
                "bytes_sent": net_io.bytes_sent,
                "bytes_recv": net_io.bytes_recv,
                "packets_sent": net_io.packets_sent,
                "packets_recv": net_io.packets_recv
            }
        
        # Get GPU info if available
        try:
            import GPUtil
            gpus = GPUtil.getGPUs()
            if gpus:
                gpu = gpus[0]  # First GPU
                metrics["gpu"] = {
                    "name": gpu.name,
                    "memory_used": gpu.memoryUsed,
                    "memory_total": gpu.memoryTotal,
                    "memory_percent": (gpu.memoryUsed / gpu.memoryTotal) * 100,
                    "temperature": gpu.temperature,
                    "load": gpu.load * 100
                }
        except ImportError:
            # GPUtil not available
            pass
        except Exception:
            # GPU not available or other error
            pass
        
        return metrics
    
    def test_inference(self) -> Dict[str, Any]:
        """Test model inference with a simple request"""
        try:
            start_time = time.time()
            response = requests.post(
                f"{self.base_url}/v1/chat/completions",
                json={
                    "model": "LFM2-1.2B",
                    "messages": [{"role": "user", "content": "Hello!"}],
                    "max_tokens": 10,
                    "temperature": 0.3
                },
                timeout=30
            )
            inference_time = time.time() - start_time
            
            if response.status_code == 200:
                result = response.json()
                return {
                    "success": True,
                    "inference_time": inference_time,
                    "response": result["choices"][0]["message"]["content"],
                    "usage": result.get("usage", {}),
                    "tokens_per_second": result.get("usage", {}).get("completion_tokens", 0) / inference_time if inference_time > 0 else 0
                }
            else:
                return {
                    "success": False,
                    "inference_time": inference_time,
                    "error": f"HTTP {response.status_code}: {response.text}"
                }
        except Exception as e:
            return {
                "success": False,
                "inference_time": 0.0,
                "error": str(e)
            }
    
    def collect_metrics(self) -> VLLMMetrics:
        """Collect comprehensive metrics"""
        health = self.check_health()
        system = self.get_system_metrics()
        stats = self.get_server_stats()
        
        return VLLMMetrics(
            timestamp=datetime.now(),
            status=health["status"],
            response_time=health["response_time"],
            memory_usage=system["memory"]["percent"],
            cpu_usage=system["cpu_percent"],
            gpu_memory=system.get("gpu", {}).get("memory_percent"),
            active_requests=stats.get("num_requests_running", 0),
            completed_requests=stats.get("num_requests_finished", 0),
            error_count=stats.get("num_requests_errored", 0)
        )
    
    def save_metrics(self, filename: str = "/var/log/vllm-metrics.jsonl"):
        """Save metrics to file"""
        try:
            with open(filename, "a") as f:
                for metric in self.metrics_history[-10:]:  # Save last 10 metrics
                    data = {
                        "timestamp": metric.timestamp.isoformat(),
                        "status": metric.status,
                        "response_time": metric.response_time,
                        "memory_usage": metric.memory_usage,
                        "cpu_usage": metric.cpu_usage,
                        "gpu_memory": metric.gpu_memory,
                        "active_requests": metric.active_requests,
                        "completed_requests": metric.completed_requests,
                        "error_count": metric.error_count
                    }
                    f.write(json.dumps(data) + "\n")
            
            # Clear saved metrics from memory
            self.metrics_history = self.metrics_history[-10:]
        except Exception as e:
            print(f"Failed to save metrics: {e}")
    
    def print_status(self, detailed: bool = False):
        """Print current status"""
        print("\n" + "="*80)
        print(f"vLLM Server Status - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print("="*80)
        
        # Health check
        health = self.check_health()
        status_color = "\033[92m" if health["status"] == "healthy" else "\033[91m"
        print(f"Status: {status_color}{health['status'].upper()}\033[0m")
        print(f"Response Time: {health['response_time']:.3f}s")
        
        if health["status"] != "healthy":
            print(f"Error: {health.get('error', 'Unknown')}")
            return
        
        # Models
        models = self.get_models()
        if "data" in models:
            print(f"Available Models: {len(models['data'])}")
            for model in models["data"]:
                print(f"  - {model['id']}")
        
        # System metrics
        system = self.get_system_metrics()
        print(f"\nSystem Metrics:")
        print(f"  CPU Usage: {system['cpu_percent']:.1f}%")
        print(f"  Memory Usage: {system['memory']['percent']:.1f}%")
        print(f"  Disk Usage: {system['disk']['percent']:.1f}%")
        
        if "gpu" in system:
            gpu = system["gpu"]
            print(f"  GPU ({gpu['name']}): {gpu['memory_percent']:.1f}% memory, {gpu['load']:.1f}% load")
        
        # Server stats
        stats = self.get_server_stats()
        if stats:
            print(f"\nServer Stats:")
            print(f"  Running Requests: {stats.get('num_requests_running', 'N/A')}")
            print(f"  Completed Requests: {stats.get('num_requests_finished', 'N/A')}")
            print(f"  Error Count: {stats.get('num_requests_errored', 'N/A')}")
        
        if detailed:
            # Test inference
            print(f"\nTesting Inference...")
            test = self.test_inference()
            if test["success"]:
                print(f"  ✓ Inference Time: {test['inference_time']:.3f}s")
                print(f"  ✓ Tokens/sec: {test['tokens_per_second']:.1f}")
                print(f"  ✓ Response: {test['response'][:50]}...")
            else:
                print(f"  ✗ Inference Failed: {test['error']}")
    
    def monitor_continuous(self, interval: int = 30, save_interval: int = 300):
        """Monitor continuously with specified interval"""
        print(f"Starting continuous monitoring (interval: {interval}s)")
        print("Press Ctrl+C to stop")
        
        last_save = time.time()
        
        while self.running:
            try:
                # Collect metrics
                metrics = self.collect_metrics()
                self.metrics_history.append(metrics)
                
                # Keep history manageable
                if len(self.metrics_history) > self.max_history:
                    self.metrics_history = self.metrics_history[-self.max_history:]
                
                # Print current status
                self.print_status()
                
                # Save metrics periodically
                if time.time() - last_save > save_interval:
                    self.save_metrics()
                    last_save = time.time()
                
                # Wait for next interval
                time.sleep(interval)
                
            except KeyboardInterrupt:
                break
            except Exception as e:
                print(f"Error during monitoring: {e}")
                time.sleep(5)
        
        # Save final metrics
        self.save_metrics()
        print("\nMonitoring stopped.")

def main():
    """Main function"""
    parser = argparse.ArgumentParser(description="vLLM Server Monitor for LFM2-1.2B")
    parser.add_argument("--host", default="127.0.0.1", help="vLLM server host")
    parser.add_argument("--port", type=int, default=8000, help="vLLM server port")
    
    subparsers = parser.add_subparsers(dest="command", help="Available commands")
    
    # Status command
    status_parser = subparsers.add_parser("status", help="Show current status")
    status_parser.add_argument("--detailed", action="store_true", help="Show detailed status")
    
    # Monitor command
    monitor_parser = subparsers.add_parser("monitor", help="Continuous monitoring")
    monitor_parser.add_argument("--interval", type=int, default=30, help="Monitoring interval in seconds")
    monitor_parser.add_argument("--save-interval", type=int, default=300, help="Metrics save interval in seconds")
    
    # Test command
    test_parser = subparsers.add_parser("test", help="Test inference")
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return
    
    monitor = VLLMMonitor(args.host, args.port)
    
    if args.command == "status":
        monitor.print_status(args.detailed)
    
    elif args.command == "monitor":
        monitor.monitor_continuous(args.interval, args.save_interval)
    
    elif args.command == "test":
        test_result = monitor.test_inference()
        print(json.dumps(test_result, indent=2))

if __name__ == "__main__":
    main()