#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Ensure jq is installed
if ! command -v jq &> /dev/null; then
  echo "jq is required for this script to run. Please install jq and try again."
  exit 1
fi

# Check if package.json exists in the root directory
if [ ! -f package.json ]; then
  echo "Root package.json not found!"
  exit 1
fi

# Read the pnpm.overrides from the root package.json
overrides=$(jq -r '.["pnpm"]["overrides"]' package.json)

# Debugging: Print the overrides
echo "pnpm.overrides:"
echo "$overrides"

# Check if pnpm.overrides is present and not null
if [ "$overrides" == "null" ] || [ -z "$overrides" ]; then
  echo "pnpm.overrides field not found or is empty in the root package.json!"
  exit 1
fi

# Find all package.json files recursively in the repository, excluding node_modules folders
package_json_files=$(find . -name "node_modules" -prune -o -name "package.json" -print)

# Iterate through each package.json file found
for file in $package_json_files; do
  # Read dependencies, devDependencies, and peerDependencies from the current package.json
  dependencies=$(jq -r '.dependencies' "$file")
  devDependencies=$(jq -r '.devDependencies' "$file")
  peerDependencies=$(jq -r '.peerDependencies' "$file")

  # Update dependencies with overrides if they exist
  if [ "$dependencies" != "null" ]; then
    for package in $(echo "$dependencies" | jq -r 'keys[]'); do
      version=$(echo "$overrides" | jq -r --arg package "$package" '.[$package]')
      if [ "$version" != "null" ]; then
        dependencies=$(echo "$dependencies" | jq --arg package "$package" --arg version "$version" '.[$package] = $version')
      fi
    done
  fi

  # Update devDependencies with overrides if they exist
  if [ "$devDependencies" != "null" ]; then
    for package in $(echo "$devDependencies" | jq -r 'keys[]'); do
      version=$(echo "$overrides" | jq -r --arg package "$package" '.[$package]')
      if [ "$version" != "null" ]; then
        devDependencies=$(echo "$devDependencies" | jq --arg package "$package" --arg version "$version" '.[$package] = $version')
      fi
    done
  fi

  # Update peerDependencies with overrides if they exist
  if [ "$peerDependencies" != "null" ]; then
    for package in $(echo "$peerDependencies" | jq -r 'keys[]'); do
      version=$(echo "$overrides" | jq -r --arg package "$package" '.[$package]')
      if [ "$version" != "null" ]; then
        peerDependencies=$(echo "$peerDependencies" | jq --arg package "$package" --arg version "$version" '.[$package] = $version')
      fi
    done
  fi

  # Write updated dependencies back to the current package.json
  jq --argjson dependencies "$dependencies" \
     --argjson devDependencies "$devDependencies" \
     --argjson peerDependencies "$peerDependencies" \
     'if $dependencies != null then .dependencies = $dependencies else . end |
      if $devDependencies != null then .devDependencies = $devDependencies else . end |
      if $peerDependencies != null then .peerDependencies = $peerDependencies else . end' \
     "$file" > "$file.tmp" && mv "$file.tmp" "$file"

  echo "Updated $file"
done

echo "All package.json files have been updated."
