#!/usr/bin/env python3
"""
Deploy vLLM setup to remote PMA server
Deploys all vLLM components to pma@192.168.100.1
"""

import subprocess
import sys
import os
from pathlib import Path

class VLLMDeployer:
    """Deploy vLLM components to remote server"""
    
    def __init__(self):
        self.remote_host = "pma@192.168.100.1"
        self.scripts_dir = Path(__file__).parent
        self.remote_scripts_dir = "/opt/pma/scripts"
        
    def run_command(self, cmd: list, description: str) -> bool:
        """Run a command and return success status"""
        print(f"📋 {description}")
        try:
            result = subprocess.run(cmd, check=True, capture_output=True, text=True)
            if result.stdout:
                print(f"   {result.stdout.strip()}")
            return True
        except subprocess.CalledProcessError as e:
            print(f"❌ Failed: {e}")
            if e.stderr:
                print(f"   Error: {e.stderr.strip()}")
            return False
    
    def deploy_scripts(self) -> bool:
        """Deploy vLLM scripts to remote server"""
        scripts = [
            "vllm-service.py",
            "vllm-monitor.py", 
            "vllm.service",
            "setup-vllm.sh"
        ]
        
        success = True
        for script in scripts:
            script_path = self.scripts_dir / script
            if not script_path.exists():
                print(f"❌ Script not found: {script_path}")
                success = False
                continue
            
            # Copy script to remote
            cmd = [
                "scp", str(script_path), 
                f"{self.remote_host}:/tmp/{script}"
            ]
            
            if not self.run_command(cmd, f"Copying {script}"):
                success = False
        
        return success
    
    def setup_remote(self) -> bool:
        """Run setup on remote server"""
        commands = [
            # Create directories and move scripts
            "sudo mkdir -p /opt/pma/scripts",
            "sudo mv /tmp/vllm-service.py /opt/pma/scripts/",
            "sudo mv /tmp/vllm-monitor.py /opt/pma/scripts/",
            "sudo mv /tmp/vllm.service /opt/pma/scripts/",
            "sudo mv /tmp/setup-vllm.sh /opt/pma/scripts/",
            
            # Set permissions
            "sudo chmod +x /opt/pma/scripts/setup-vllm.sh",
            "sudo chmod +x /opt/pma/scripts/vllm-service.py", 
            "sudo chmod +x /opt/pma/scripts/vllm-monitor.py",
            
            # Run setup script
            "sudo /opt/pma/scripts/setup-vllm.sh",
        ]
        
        for cmd_str in commands:
            cmd = ["ssh", self.remote_host, cmd_str]
            if not self.run_command(cmd, f"Remote: {cmd_str}"):
                return False
        
        return True
    
    def deploy_backend_config(self) -> bool:
        """Deploy updated backend configuration"""
        config_path = self.scripts_dir.parent / "configs" / "config.yaml"
        if not config_path.exists():
            print(f"❌ Config not found: {config_path}")
            return False
        
        # Copy config
        cmd = [
            "scp", str(config_path),
            f"{self.remote_host}:/opt/pma/backend/configs/config.yaml"
        ]
        
        return self.run_command(cmd, "Deploying backend configuration")
    
    def restart_backend(self) -> bool:
        """Restart PMA backend to pick up new config"""
        cmd = ["ssh", self.remote_host, "sudo systemctl restart pma-backend"]
        return self.run_command(cmd, "Restarting PMA backend")
    
    def check_status(self) -> bool:
        """Check the status of deployed services"""
        print("\n🔍 Checking service status...")
        
        commands = [
            ("Backend status", "sudo systemctl status pma-backend --no-pager -l"),
            ("vLLM service status", "sudo systemctl status vllm --no-pager -l"),
            ("vLLM health check", "sudo -u pma /opt/pma/scripts/vllm-monitor.py status"),
        ]
        
        for description, cmd_str in commands:
            cmd = ["ssh", self.remote_host, cmd_str]
            print(f"\n📋 {description}")
            try:
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
                if result.stdout:
                    print(result.stdout)
                if result.stderr:
                    print(f"Error: {result.stderr}")
            except subprocess.TimeoutExpired:
                print("⏰ Command timed out")
            except Exception as e:
                print(f"❌ Error: {e}")
        
        return True
    
    def deploy(self) -> bool:
        """Full deployment process"""
        print("🚀 Starting vLLM deployment to PMA server")
        print(f"   Target: {self.remote_host}")
        print(f"   Model: LiquidAI/LFM2-1.2B")
        print()
        
        steps = [
            ("Deploy scripts", self.deploy_scripts),
            ("Setup remote services", self.setup_remote),
            ("Deploy backend config", self.deploy_backend_config),
            ("Restart backend", self.restart_backend),
        ]
        
        for step_name, step_func in steps:
            print(f"\n🔄 {step_name}...")
            if not step_func():
                print(f"❌ Failed at step: {step_name}")
                return False
            print(f"✅ {step_name} completed")
        
        # Check final status
        self.check_status()
        
        print("\n🎉 vLLM deployment completed!")
        print("\nNext steps:")
        print("1. Start vLLM service:")
        print(f"   ssh {self.remote_host} 'vllm-ctl start'")
        print("\n2. Monitor status:")
        print(f"   ssh {self.remote_host} 'vllm-ctl health'")
        print("\n3. Test inference:")
        print(f"   ssh {self.remote_host} 'vllm-ctl test'")
        print("\n4. View logs:")
        print(f"   ssh {self.remote_host} 'vllm-ctl logs'")
        
        return True

def main():
    """Main function"""
    if len(sys.argv) > 1 and sys.argv[1] == "--status-only":
        deployer = VLLMDeployer()
        deployer.check_status()
        return
    
    deployer = VLLMDeployer()
    success = deployer.deploy()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()