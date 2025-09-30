#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Function to read the version from the VERSION file
get_version() {
    local version_file="../core/pkg/version/VERSION"
    if [[ -f "$version_file" ]]; then
        # Extract the major.minor part of the version
        VERSION=$(cat "$version_file" | cut -d '.' -f1-2)
        echo "Current version is ${VERSION}.x"
    else
        echo "VERSION file not found at $version_file!"
        exit 1
    fi
}

# Function to check the version in MODULE.bazel
check_module_bazel() {
    local version=$1
    local module_file="../MODULE.bazel"

    if [[ -f "$module_file" ]]; then
        # Extract version from within the module() declaration
        bazel_version=$(awk '/^module\(/,/^\)/ {if (/version = /) print}' "$module_file" | cut -d '"' -f2 | cut -d '.' -f1-2)
        if [[ "$bazel_version" == "$version" ]]; then
            echo "Version match in MODULE.bazel"
            return 0
        else
            echo "Version mismatch in MODULE.bazel: found $bazel_version, expected $version"
            return 1
        fi
    else
        echo "MODULE.bazel not found!"
        return 1
    fi
}

# Function to check the version in console/src-tauri/Cargo.toml
check_cargo_toml() {
    local version=$1
    local cargo_file="../console/src-tauri/Cargo.toml"

    if [[ -f "$cargo_file" ]]; then
        cargo_version=$(grep '^version = ' "$cargo_file" | head -1 | cut -d '"' -f2 | cut -d '.' -f1-2)
        if [[ "$cargo_version" == "$version" ]]; then
            echo "Version match in console/src-tauri/Cargo.toml"
            return 0
        else
            echo "Version mismatch in console/src-tauri/Cargo.toml: found $cargo_version, expected $version"
            return 1
        fi
    else
        echo "console/src-tauri/Cargo.toml not found!"
        return 1
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

paths=("../x/ts" "../x/media" "../alamos/ts" "../client/ts" "../pluto" "../console" "../drift" ".." "../freighter/ts" "../freighter/py" "../alamos/py" "../client/py" "../integration")

version_check_passed=true

# Check MODULE.bazel
check_module_bazel "$VERSION"
if [[ $? -ne 0 ]]; then
    version_check_passed=false
fi

# Check console/src-tauri/Cargo.toml
check_cargo_toml "$VERSION"
if [[ $? -ne 0 ]]; then
    version_check_passed=false
fi

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
