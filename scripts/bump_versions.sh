#!/bin/bash

# Check if a version argument is provided
if [[ -z "$1" ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

VERSION=$1

# Function to update the version in the VERSION file
update_version_file() {
    local version_file="../synnax/pkg/version/VERSION"
    echo "$VERSION" > "$version_file"
    echo "Updated VERSION file to $VERSION"
}

# Function to update the version in a Python pyproject.toml file
update_python_version() {
    local path=$1
    local version=$2
    if [[ -f "$path/pyproject.toml" ]]; then
        sed -i '' "s/^version = \".*\"/version = \"$version\"/" "$path/pyproject.toml"
        echo "Updated version in $path/pyproject.toml"
    else
        echo "pyproject.toml not found in $path!"
    fi
}

# Function to update the version in a TypeScript package.json file
update_typescript_version() {
    local path=$1
    local version=$2
    if [[ -f "$path/package.json" ]]; then
        sed -i '' "s/\"version\": \".*\"/\"version\": \"$version\"/" "$path/package.json"
        echo "Updated version in $path/package.json"
    else
        echo "package.json not found in $path!"
    fi
}

# Main script execution
update_version_file

# Arrays for paths
typescript_paths=("../x/ts" "../alamos/ts" "../client/ts" "../pluto" "../console" "../drift" ".." "../freighter/ts")
python_paths=("../freighter/py" "../alamos/py" "../client/py")

# Update TypeScript versions
for path in "${typescript_paths[@]}"; do
    update_typescript_version "$path" "$VERSION"
done

# Update Python versions
for path in "${python_paths[@]}"; do
    update_python_version "$path" "$VERSION"
done

echo "Version update complete."

# Call the check_version.sh script to verify the updates
./check_version.sh

# Capture the exit status of check_version.sh
CHECK_VERSION_STATUS=$?

# Print a message based on the status
if [[ $CHECK_VERSION_STATUS -eq 0 ]]; then
    echo "All version checks passed successfully."
else
    echo "Some version checks failed. Please review the output above."
fi

# Exit with the status of check_version.sh
exit $CHECK_VERSION_STATUS
