#!/bin/bash

echo "Host confirmed"
echo "OS: $(uname -s)"
echo "Kernel: $(uname -r)"
echo "Architecture: $(uname -m)"
echo "Hostname: $(hostname)"

if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    echo "CPU cores: $(sysctl -n hw.ncpu)"
    echo "Physical memory: $(( $(sysctl -n hw.memsize) / 1024 / 1024 / 1024 ))GB"
    echo "macOS version: $(sw_vers -productVersion)"
else
    # Linux
    echo "CPU cores: $(nproc)"
    echo "Memory: $(free -h | grep '^Mem:' | awk '{print $2}')"
fi

echo "Disk space: $(df -h / | tail -1 | awk '{print $4}') available"
echo "Date: $(date)"