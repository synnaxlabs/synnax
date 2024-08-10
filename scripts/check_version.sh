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

# Function to check the version in a Python pyproject.toml file
check_python_version() {
    local path=$1
    local version=$2
    if [[ -f "$path/pyproject.toml" ]]; then
        py_version=$(grep '^version = ' "$path/pyproject.toml" | cut -d '"' -f2)
        if [[ "$py_version" == "$version" ]]; then
            echo "Version match in $path/pyproject.toml"
            return 0
        else
            echo "Version mismatch in $path/pyproject.toml: found $py_version, expected $version"
            return 1
        fi
    else
        echo "pyproject.toml not found in $path!"
        return 1
    fi
}

# Function to check the version in a TypeScript package.json file
check_typescript_version() {
    local path=$1
    local version=$2
    if [[ -f "$path/package.json" ]]; then
        ts_version=$(grep '"version": ' "$path/package.json" | cut -d '"' -f4)
        if [[ "$ts_version" == "$version" ]]; then
            echo "Version match in $path/package.json"
            return 0
        else
            echo "Version mismatch in $path/package.json: found $ts_version, expected $version"
            return 1
        fi
    else
        echo "package.json not found in $path!"
        return 1
    fi
}

# Main script execution
get_version

# Arrays for paths and results
typescript_paths=("../x/ts" "../alamos/ts" "../client/ts" "../pluto" "../console" "../drift" ".." "../freighter/ts")
python_paths=("../freighter/py" "../alamos/py" "../client/py")

# Check TypeScript versions
typescript_results=()
for path in "${typescript_paths[@]}"; do
    check_typescript_version "$path" "$VERSION"
    typescript_results+=($?)
done

# Check Python versions
python_results=()
for path in "${python_paths[@]}"; do
    check_python_version "$path" "$VERSION"
    python_results+=($?)
done

# Exit with status code 1 if any version check fails
for result in "${typescript_results[@]}" "${python_results[@]}"; do
    if [[ $result -ne 0 ]]; then
        echo "Version check failed."
        exit 1
    fi
done

echo "All versions match."
exit 0
