#!/bin/bash
#
# Setup script for vLLM with LiquidAI LFM2-1.2B
# Based on specifications from https://huggingface.co/LiquidAI/LFM2-1.2B
#
set -euo pipefail

# Configuration
VENV_PATH="/opt/pma/backend/.venv"
SERVICE_USER="pma"
SERVICE_GROUP="pma"
LOG_DIR="/var/log"
SCRIPTS_DIR="/opt/pma/scripts"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check if venv exists
check_venv() {
    if [[ ! -d "$VENV_PATH" ]]; then
        log_error "Virtual environment not found at $VENV_PATH"
        log_info "Please create the virtual environment first"
        exit 1
    fi
    
    if [[ ! -f "$VENV_PATH/bin/python" ]]; then
        log_error "Python executable not found in $VENV_PATH"
        exit 1
    fi
    
    log_success "Virtual environment found at $VENV_PATH"
}

# Install vLLM in the virtual environment
install_vllm() {
    log_info "Installing vLLM in virtual environment..."
    
    # Activate virtual environment and install vLLM
    source "$VENV_PATH/bin/activate"
    
    # Upgrade pip first
    log_info "Upgrading pip..."
    python -m pip install --upgrade pip
    
    # Install vLLM with specific optimizations for LFM2
    log_info "Installing vLLM (this may take several minutes)..."
    python -m pip install vllm
    
    # Install additional dependencies for monitoring
    log_info "Installing monitoring dependencies..."
    python -m pip install psutil requests
    
    # Try to install GPU monitoring (optional)
    if command -v nvidia-smi &> /dev/null; then
        log_info "NVIDIA GPU detected, installing GPU monitoring..."
        python -m pip install gpustat GPUtil || log_warning "Failed to install GPU monitoring (optional)"
    fi
    
    # Verify vLLM installation
    log_info "Verifying vLLM installation..."
    if python -c "import vllm; print(f'vLLM version: {vllm.__version__}')" 2>/dev/null; then
        log_success "vLLM installed successfully"
    else
        log_error "vLLM installation verification failed"
        exit 1
    fi
    
    deactivate
}

# Setup directories and permissions
setup_directories() {
    log_info "Setting up directories and permissions..."
    
    # Create scripts directory
    mkdir -p "$SCRIPTS_DIR"
    
    # Copy scripts to the target directory
    if [[ -f "$(dirname "$0")/vllm-service.py" ]]; then
        cp "$(dirname "$0")/vllm-service.py" "$SCRIPTS_DIR/"
        chmod +x "$SCRIPTS_DIR/vllm-service.py"
        log_success "Copied vllm-service.py to $SCRIPTS_DIR"
    fi
    
    if [[ -f "$(dirname "$0")/vllm-monitor.py" ]]; then
        cp "$(dirname "$0")/vllm-monitor.py" "$SCRIPTS_DIR/"
        chmod +x "$SCRIPTS_DIR/vllm-monitor.py"
        log_success "Copied vllm-monitor.py to $SCRIPTS_DIR"
    fi
    
    # Set ownership
    chown -R "$SERVICE_USER:$SERVICE_GROUP" "$SCRIPTS_DIR"
    
    # Create log directory permissions
    touch "$LOG_DIR/vllm-service.log"
    touch "$LOG_DIR/vllm-metrics.jsonl"
    chown "$SERVICE_USER:$SERVICE_GROUP" "$LOG_DIR/vllm-service.log"
    chown "$SERVICE_USER:$SERVICE_GROUP" "$LOG_DIR/vllm-metrics.jsonl"
    chmod 644 "$LOG_DIR/vllm-service.log" "$LOG_DIR/vllm-metrics.jsonl"
}

# Install systemd service
install_systemd_service() {
    log_info "Installing systemd service..."
    
    # Copy service file
    if [[ -f "$(dirname "$0")/vllm.service" ]]; then
        cp "$(dirname "$0")/vllm.service" /etc/systemd/system/
        chmod 644 /etc/systemd/system/vllm.service
        log_success "Copied vllm.service to /etc/systemd/system/"
    else
        log_error "vllm.service file not found"
        exit 1
    fi
    
    # Reload systemd
    systemctl daemon-reload
    log_success "Systemd daemon reloaded"
    
    # Enable service (but don't start yet)
    systemctl enable vllm.service
    log_success "vLLM service enabled"
}

# Download and cache LFM2 model
download_model() {
    log_info "Pre-downloading LiquidAI/LFM2-1.2B model..."
    log_info "This may take several minutes depending on your internet connection..."
    
    # Run as the service user
    sudo -u "$SERVICE_USER" bash -c "
        source '$VENV_PATH/bin/activate'
        python -c \"
from transformers import AutoTokenizer, AutoModelForCausalLM
import os

# Set cache directory
os.environ['HF_HOME'] = '/opt/pma/models/huggingface'
os.makedirs('/opt/pma/models/huggingface', exist_ok=True)

print('Downloading tokenizer...')
tokenizer = AutoTokenizer.from_pretrained('LiquidAI/LFM2-1.2B')
print('Tokenizer downloaded successfully')

print('Downloading model (this will take a while)...')
model = AutoModelForCausalLM.from_pretrained(
    'LiquidAI/LFM2-1.2B',
    torch_dtype='bfloat16',
    device_map='auto'
)
print('Model downloaded successfully')
print('Model size: {:.2f}B parameters'.format(sum(p.numel() for p in model.parameters()) / 1e9))
\"
    "
    
    # Set ownership of model cache
    chown -R "$SERVICE_USER:$SERVICE_GROUP" /opt/pma/models/
    
    log_success "LFM2-1.2B model downloaded and cached"
}

# Test vLLM installation
test_vllm() {
    log_info "Testing vLLM installation..."
    
    # Run a basic test as the service user
    sudo -u "$SERVICE_USER" bash -c "
        source '$VENV_PATH/bin/activate'
        python -c \"
import vllm
print('vLLM version:', vllm.__version__)

# Test model loading (just check if it can be imported)
from vllm import LLM
print('vLLM LLM class imported successfully')
\"
    "
    
    log_success "vLLM installation test passed"
}

# Create management aliases
create_aliases() {
    log_info "Creating management aliases..."
    
    # Create convenient aliases for vLLM management
    cat > /usr/local/bin/vllm-ctl << 'EOF'
#!/bin/bash
# vLLM Control Script

SCRIPTS_DIR="/opt/pma/scripts"

case "$1" in
    start)
        sudo systemctl start vllm
        ;;
    stop)
        sudo systemctl stop vllm
        ;;
    restart)
        sudo systemctl restart vllm
        ;;
    status)
        sudo systemctl status vllm
        ;;
    logs)
        sudo journalctl -u vllm -f
        ;;
    monitor)
        sudo -u pma python3 "$SCRIPTS_DIR/vllm-monitor.py" monitor
        ;;
    test)
        sudo -u pma python3 "$SCRIPTS_DIR/vllm-monitor.py" test
        ;;
    health)
        sudo -u pma python3 "$SCRIPTS_DIR/vllm-monitor.py" status --detailed
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|monitor|test|health}"
        echo ""
        echo "Commands:"
        echo "  start   - Start vLLM service"
        echo "  stop    - Stop vLLM service"
        echo "  restart - Restart vLLM service"
        echo "  status  - Show service status"
        echo "  logs    - Show live logs"
        echo "  monitor - Start continuous monitoring"
        echo "  test    - Test model inference"
        echo "  health  - Show detailed health status"
        exit 1
        ;;
esac
EOF

    chmod +x /usr/local/bin/vllm-ctl
    log_success "Created vllm-ctl command"
}

# Create logrotate configuration
setup_logrotate() {
    log_info "Setting up log rotation..."
    
    cat > /etc/logrotate.d/vllm << 'EOF'
/var/log/vllm-service.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    copytruncate
    create 644 pma pma
}

/var/log/vllm-metrics.jsonl {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    copytruncate
    create 644 pma pma
}
EOF
    
    log_success "Log rotation configured"
}

# Print final instructions
print_instructions() {
    log_success "vLLM setup completed successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Enable vLLM in PMA configuration:"
    echo "   Edit /opt/pma/backend/configs/config.yaml:"
    echo "   Set ai.vllm.enabled: true"
    echo ""
    echo "2. Start vLLM service:"
    echo "   vllm-ctl start"
    echo ""
    echo "3. Monitor service:"
    echo "   vllm-ctl status"
    echo "   vllm-ctl health"
    echo ""
    echo "4. Test inference:"
    echo "   vllm-ctl test"
    echo ""
    echo "5. View logs:"
    echo "   vllm-ctl logs"
    echo ""
    echo "Available commands: vllm-ctl {start|stop|restart|status|logs|monitor|test|health}"
    echo ""
    echo "Model: LiquidAI/LFM2-1.2B"
    echo "Server: http://127.0.0.1:8000"
    echo "API: OpenAI-compatible"
    echo ""
}

# Main execution
main() {
    log_info "Starting vLLM setup for LiquidAI LFM2-1.2B..."
    log_info "Based on: https://huggingface.co/LiquidAI/LFM2-1.2B"
    echo ""
    
    check_root
    check_venv
    install_vllm
    setup_directories
    install_systemd_service
    download_model
    test_vllm
    create_aliases
    setup_logrotate
    
    print_instructions
}

# Run main function
main "$@"