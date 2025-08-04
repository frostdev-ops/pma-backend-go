#!/bin/bash
# vLLM Control Script for LiquidAI LFM2-1.2B

SCRIPTS_DIR="/opt/pma/scripts"

case "$1" in
    start)
        echo "🚀 Starting vLLM service..."
        sudo systemctl start vllm
        ;;
    stop)
        echo "⏹️ Stopping vLLM service..."
        sudo systemctl stop vllm
        ;;
    restart)
        echo "🔄 Restarting vLLM service..."
        sudo systemctl restart vllm
        ;;
    status)
        sudo systemctl status vllm --no-pager
        ;;
    logs)
        echo "📋 vLLM service logs (Ctrl+C to exit):"
        sudo journalctl -u vllm -f
        ;;
    monitor)
        echo "📊 Starting vLLM monitoring (Ctrl+C to exit):"
        sudo -u pma python3 "$SCRIPTS_DIR/vllm-monitor.py" monitor
        ;;
    test)
        echo "🧪 Testing vLLM inference..."
        sudo -u pma python3 "$SCRIPTS_DIR/vllm-monitor.py" test
        ;;
    health)
        echo "🏥 vLLM health check..."
        sudo -u pma python3 "$SCRIPTS_DIR/vllm-monitor.py" status --detailed
        ;;
    download)
        echo "📥 Pre-downloading LFM2 GGUF model..."
        sudo -u pma bash -c "
            source /opt/pma/backend/.venv/bin/activate
            export HF_HOME=/opt/pma/models/huggingface
            mkdir -p /opt/pma/models/huggingface
            python -c \"
from huggingface_hub import hf_hub_download
import os

# Set cache directory
os.environ['HF_HOME'] = '/opt/pma/models/huggingface'

print('Downloading LiquidAI/LFM2-1.2B-GGUF quantized model...')
try:
    # Download the GGUF file
    file_path = hf_hub_download(
        repo_id='LiquidAI/LFM2-1.2B-GGUF',
        filename='LFM2-1.2B-Q4_0.gguf',
        cache_dir='/opt/pma/models/huggingface'
    )
    print(f'GGUF model downloaded successfully to: {file_path}')
    
    # Also download tokenizer from base model
    from transformers import AutoTokenizer
    tokenizer = AutoTokenizer.from_pretrained('LiquidAI/LFM2-1.2B')
    print('Tokenizer downloaded successfully!')
    
except Exception as e:
    print(f'Error downloading model: {e}')
\"
        "
        ;;
    *)
        echo "vLLM Control - LiquidAI LFM2-1.2B Management"
        echo ""
        echo "Usage: $0 {start|stop|restart|status|logs|monitor|test|health|download}"
        echo ""
        echo "Commands:"
        echo "  start    - Start vLLM service"
        echo "  stop     - Stop vLLM service"
        echo "  restart  - Restart vLLM service"
        echo "  status   - Show service status"
        echo "  logs     - Show live logs"
        echo "  monitor  - Start continuous monitoring"
        echo "  test     - Test model inference"
        echo "  health   - Show detailed health status"
        echo "  download - Pre-download LFM2 model"
        echo ""
        echo "Model: LiquidAI/LFM2-1.2B"
        echo "Server: http://127.0.0.1:8000"
        echo "API: OpenAI-compatible"
        exit 1
        ;;
esac