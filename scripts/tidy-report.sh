#!/bin/bash

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Column widths
FILE_WIDTH=35
TARGET_WIDTH=15
DIAG_WIDTH=45
LINE_WIDTH=6
TYPE_WIDTH=12

# Function to extract target name
get_target() {
    local file="$1"
    echo "$file" | sed -E 's/.*\.([^.]+)\.clang-tidy\.yaml/\1/'
}

# Function to clean up filename
clean_filename() {
    local file="$1"
    # Then get just the filename without the path and preserve original extension
    basename "$file" | sed -E 's/\.([^.]+)\.clang-tidy\.yaml//'
}

# Function to process a single YAML file
process_yaml() {
    local file="$1"
    local source_file=$(clean_filename "$file")
    local target=$(get_target "$file")
    
    # Extract diagnostics using grep and sed
    while IFS= read -r line; do
        if [[ $line =~ "DiagnosticName:" ]]; then
            diagnostic_name=$(echo "$line" | sed 's/.*DiagnosticName: *\(.*\)/\1/')
        elif [[ $line =~ "Message:" ]]; then
            message=$(echo "$line" | sed "s/.*Message: *'\(.*\)'/\1/")
        elif [[ $line =~ "FileOffset:" ]]; then
            file_offset=$(echo "$line" | grep -o '[0-9]\+')
        elif [[ $line =~ "Level:" ]]; then
            level=$(echo "$line" | sed 's/.*Level: *\(.*\)/\1/')
            # Print when we have all parts of a diagnostic
            if [[ -n "$diagnostic_name" && -n "$message" && -n "$file_offset" && -n "$level" ]]; then
                printf "%-${FILE_WIDTH}s | %-${TARGET_WIDTH}s | %-${DIAG_WIDTH}s | %${LINE_WIDTH}s | %-${TYPE_WIDTH}s | %s\n" \
                    "$source_file" "$target" "$diagnostic_name" "$file_offset" "$level" "$message"
                # Reset variables
                diagnostic_name=""
                message=""
                file_offset=""
                level=""
            fi
        fi
    done < "$file"
}

# Print header
print_header() {
    echo -e "${YELLOW}"
    printf "%-${FILE_WIDTH}s | %-${TARGET_WIDTH}s | %-${DIAG_WIDTH}s | %${LINE_WIDTH}s | %-${TYPE_WIDTH}s | %s\n" \
        "FILE" "TARGET" "DIAGNOSTIC" "LINE" "TYPE" "MESSAGE"
    printf "%s\n" "$(printf '=%.0s' {1..150})"
    echo -e "${NC}"
}

# Main script
main() {
    local dir="${1:-.}"  # Use current directory if none specified
    
    # Convert bazel-bin path to bazel target
    local target
    if [[ "$dir" == "bazel-bin/driver" ]]; then
        target="//driver/..."
    elif [[ "$dir" == "bazel-bin/driver/ni" ]]; then
        target="//driver/ni/..."
    else
        target="//..."
    fi
    echo "Building target: $target"
    bazel build --config=clang-tidy "$target"
    
    # Now find all YAML files
    local yaml_files=$(find "$dir" -name "*.clang-tidy.yaml" 2>/dev/null)
    
    if [[ -z "$yaml_files" ]]; then
        echo -e "${RED}No tidy files generated!${NC}"
        exit 1
    fi
    
    # Print header
    print_header
    
    # Process each file
    local yaml_count=0
    local issue_count=0
    local error_count=0
    local warning_count=0
    
    while IFS= read -r file; do
        process_yaml "$file"
        # Count issues in this file
        local file_issues=$(grep -c "DiagnosticName:" "$file")
        local file_errors=$(grep -c "Error" "$file")
        local file_warnings=$(grep -c "Warning" "$file")
        ((issue_count += file_issues))
        ((error_count += file_errors))
        ((warning_count += file_warnings))
        ((yaml_count++))
    done <<< "$yaml_files"
    
    # Print summary
    echo -e "\n${YELLOW}Summary:${NC}"
    echo -e "Processed $yaml_count YAML files"
    echo -e "Total issues: $issue_count"
    echo -e "${RED}Errors: $error_count${NC}"
    echo -e "${YELLOW}Warnings: $warning_count${NC}"
}

# Run the script
main "$1"