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

# Check TypeScript versions
check_typescript_version "../x/ts" "$VERSION"
TYPESCRIPT_RESULT_1=$?

check_typescript_version "../alamos/ts" "$VERSION"
TYPESCRIPT_RESULT_2=$?

check_typescript_version "../client/ts" "$VERSION"
TYPESCRIPT_RESULT_3=$?

check_typescript_version "../pluto" "$VERSION"
TYPESCRIPT_RESULT_4=$?

check_typescript_version "../console" "$VERSION"
TYPESCRIPT_RESULT_5=$?

check_typescript_version "../drift" "$VERSION"
TYPESCRIPT_RESULT_6=$?

check_typescript_version ".." "$VERSION"
TYPESCRIPT_RESULT_7=$?

check_typescript_version "../freighter/ts" "$VERSION"
TYPESCRIPT_RESULT_8=$?

# Check Python versions
check_python_version "../freighter/py" "$VERSION"
PYTHON_RESULT_1=$?

check_python_version "../alamos/py" "$VERSION"
PYTHON_RESULT_2=$?

check_python_version "../client/py" "$VERSION"
PYTHON_RESULT_3=$?

# Exit with status code 1 if any version check fails
if [[ $TYPESCRIPT_RESULT_1 -ne 0 || $TYPESCRIPT_RESULT_2 -ne 0 || $TYPESCRIPT_RESULT_3 -ne 0 || \
      $TYPESCRIPT_RESULT_4 -ne 0 || $TYPESCRIPT_RESULT_5 -ne 0 || $TYPESCRIPT_RESULT_6 -ne 0 || \
      $TYPESCRIPT_RESULT_7 -ne 0 || $TYPESCRIPT_RESULT_8 -ne 0 || \
      $PYTHON_RESULT_1 -ne 0 || $PYTHON_RESULT_2 -ne 0 || $PYTHON_RESULT_3 -ne 0 ]]; then
    echo "Version check failed."
    exit 1
else
    echo "All versions match."
    exit 0
fi
