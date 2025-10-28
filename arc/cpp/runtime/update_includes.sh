#!/bin/bash
# Script to update #include paths after arc/cpp/runtime refactoring

cd "$(dirname "$0")"

# Update includes in all .h and .cpp files in the new directory structure
find . -type f \( -name "*.h" -o -name "*.cpp" \) | while read file; do
    # Skip if file is in parent directory (old files)
    if [[ "$file" == ./*.h ]] || [[ "$file" == ./*.cpp ]]; then
        continue
    fi

    echo "Updating includes in: $file"

    # Update paths to new locations
    sed -i '' 's|"arc/cpp/runtime/types\.h"|"arc/cpp/runtime/core/types.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/state\.h"|"arc/cpp/runtime/state/state.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/node_state\.h"|"arc/cpp/runtime/state/node_state.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/scheduler\.h"|"arc/cpp/runtime/scheduler/scheduler.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/time_wheel\.h"|"arc/cpp/runtime/scheduler/time_wheel.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/factory\.h"|"arc/cpp/runtime/factory/factory.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/runtime\.h"|"arc/cpp/runtime/wasm/runtime.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/bindings\.h"|"arc/cpp/runtime/wasm/bindings.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/module\.h"|"arc/cpp/runtime/module/loader.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/interval_factory\.h"|"arc/cpp/runtime/nodes/interval/factory.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/wasm_factory\.h"|"arc/cpp/runtime/nodes/wasm/factory.h"|g' "$file"
    sed -i '' 's|"arc/cpp/runtime/node\.h"|"arc/cpp/runtime/core/node.h"|g' "$file"
done

echo "Include path updates complete!"
