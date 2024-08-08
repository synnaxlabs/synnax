#!/bin/bash

# Function to read the version from the VERSION file
get_version() {
    local version_file="../synnax/pkg/version/VERSION"
    if [[ -f "$version_file" ]]; then
        VERSION=$(cat "$version_file")
        echo "Current version is $VERSION"
    else
        echo "VERSION file not found at $version_file!"
        exit 1
    fi
}

# Function to update the version in a Python pyproject.toml file
update_python_version() {
    local path=$1
    local version=$2
    if [[ -f "$path/pyproject.toml" ]]; then
        # Replace the version in the pyproject.toml file
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
        # Replace the version in the package.json file
        sed -i '' "s/\"version\": \".*\"/\"version\": \"$version\"/" "$path/package.json"
        echo "Updated version in $path/package.json"
    else
        echo "package.json not found in $path!"
    fi
}

# Main script execution
get_version

# Update TypeScript versions
update_typescript_version "../x/ts" "$VERSION"
update_typescript_version "../alamos/ts" "$VERSION"
update_typescript_version "../client/ts" "$VERSION"
update_typescript_version "../pluto" "$VERSION"
update_typescript_version "../console" "$VERSION"
update_typescript_version "../drift" "$VERSION"
update_typescript_version ".." "$VERSION"
update_typescript_version "../freighter/ts" "$VERSION"

# Update Python versions
update_python_version "../freighter/py" "$VERSION"
update_python_version "../alamos/py" "$VERSION"
update_python_version "../client/py" "$VERSION"

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
