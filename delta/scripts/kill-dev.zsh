set -o localoptions -o localtraps
trap 'ps aux | grep "tauri dev" | grep -v grep | awk '"'"'{print $2}'"'"' | xargs kill -9 && ps aux | grep "Synnax" | grep -v grep | awk '"'"'{print $2}'"'"' | xargs kill -9' INT
sleep 10000
echo "returned with: $?"
