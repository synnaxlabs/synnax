#!/bin/bash

# Function to read the version from the VERSION file
get_version() {
    local version_file="../synnax/pkg/version/VERSION"
    if [[ -f "$version_file" ]]; then
        # Extract the major.minor part of the version
        VERSION=$(cat "$version_file" | cut -d '.' -f1-2)
        echo "Current version is ${VERSION}.x"
    else
        echo "VERSION file not found at $version_file!"
        exit 1
    fi
}

# Combined function to check the version in either a TypeScript package.json or Python pyproject.toml file
check_version() {
    local path=$1
    local version=$2

    if [[ -f "$path/package.json" ]]; then
        ts_version=$(grep '"version": ' "$path/package.json" | cut -d '"' -f4 | cut -d '.' -f1-2)
        if [[ "$ts_version" == "$version" ]]; then
            echo "Version match in $path/package.json"
            return 0
        else
            echo "Version mismatch in $path/package.json: found $ts_version, expected $version"
            return 1
        fi
    elif [[ -f "$path/pyproject.toml" ]]; then
        py_version=$(grep '^version = ' "$path/pyproject.toml" | cut -d '"' -f2 | cut -d '.' -f1-2)
        if [[ "$py_version" == "$version" ]]; then
            echo "Version match in $path/pyproject.toml"
            return 0
        else
            echo "Version mismatch in $path/pyproject.toml: found $py_version, expected $version"
            return 1
        fi
    else
        echo "No package.json or pyproject.toml found in $path!"
        return 1
    fi
}

# Main script execution
get_version

paths=("../x/ts" "../alamos/ts" "../client/ts" "../pluto" "../console" "../drift" ".." "../freighter/ts" "../freighter/py" "../alamos/py" "../client/py")

version_check_passed=true

# Check versions in all specified paths
for path in "${paths[@]}"; do
    check_version "$path" "$VERSION"
    if [[ $? -ne 0 ]]; then
        version_check_passed=false
    fi
done

if [[ "$version_check_passed" == true ]]; then
    echo "All versions match."
    exit 0
else
    echo "Version check failed."
    exit 1
fi
