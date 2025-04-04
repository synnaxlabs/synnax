#!/bin/bash

set -e

echo "🔍 Scanning for BUILD.bazel files missing visibility flags..."

find . -name "BUILD.bazel" | while read -r file; do
  if grep -qE '(cc_library|cc_binary|cpp_grpc_library)' "$file" && ! grep -q "fvisibility=hidden" "$file"; then
    echo "🛠️  Updating: $file"

    # Insert copts = [...] after the first line that opens a rule
    awk '
      BEGIN { updated = 0 }
      /cc_library|cc_binary|cpp_grpc_library/ {
        in_rule = 1
      }
      in_rule && /^\s*\)/ {
        in_rule = 0
      }
      in_rule && /name *=/ && updated == 0 {
        print $0
        print "    copts = select({"
        print "        \"@platforms//os:windows\": [\"/std:c++20\"],"
        print "        \"//conditions:default\": ["
        print "            \"-fvisibility=hidden\","
        print "            \"-fvisibility-inlines-hidden\","
        print "        ],"
        print "    }),"
        updated = 1
        next
      }
      { print $0 }
    ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
  fi
done

echo "✅ Done. Visibility flags added where needed."