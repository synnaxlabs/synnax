#!/bin/bash

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to process a single YAML file
process_yaml() {
    local file="$1"
    local source_file=$(basename "$file" .clang-tidy.yaml)
    
    while IFS= read -r line; do
        if [[ $line =~ "DiagnosticName:" ]]; then
            diagnostic_name=$(echo "$line" | sed 's/.*DiagnosticName: *\(.*\)/\1/')
        elif [[ $line =~ "Message:" ]]; then
            message=$(echo "$line" | sed "s/.*Message: *'\(.*\)'/\1/")
        elif [[ $line =~ "FilePath:" ]]; then
            filepath=$(echo "$line" | sed "s/.*FilePath: *'\(.*\)'/\1/")
            filepath=$(basename "$filepath")
        elif [[ $line =~ "FileOffset:" ]]; then
            offset=$(echo "$line" | sed 's/.*FileOffset: *\(.*\)/\1/')
        elif [[ $line =~ "Level:" ]]; then
            level=$(echo "$line" | sed 's/.*Level: *\(.*\)/\1/')
            # Print when we have all parts of a diagnostic
            if [[ -n "$diagnostic_name" && -n "$message" && -n "$filepath" && -n "$level" ]]; then
                printf "%-30s | %-30s | %-10s | %s\n" "$filepath" "$diagnostic_name" "$level" "$message"
                # Reset variables
                diagnostic_name=""
                message=""
                filepath=""
                offset=""
                level=""
            fi
        fi
    done < "$file"
}

# Print header
print_header() {
    echo -e "${YELLOW}"
    printf "%-30s | %-30s | %-10s | %s\n" "FILE" "CHECK" "LEVEL" "MESSAGE"
    printf "%s\n" "$(printf '=%.0s' {1..120})"
    echo -e "${NC}"
}

# Main script
main() {
    local dir="${1:-.}"  # Use current directory if none specified
    
    # Find all clang-tidy YAML files
    local yaml_files=$(find "$dir" -name "*.clang-tidy.yaml" 2>/dev/null)
    
    if [[ -z "$yaml_files" ]]; then
        echo -e "${RED}No clang-tidy YAML files found!${NC}"
        exit 1
    fi

    # Print header
    print_header

    # Process each file
    local count=0
    while IFS= read -r file; do
        process_yaml "$file"
        ((count++))
    done <<< "$yaml_files"

    # Print summary
    echo -e "\n${YELLOW}Processed $count YAML files${NC}"
}

# Run the script
main "$1"