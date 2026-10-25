#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes a header from a CMake configure_file template.

Usage: configure.py <template> <output> [NAME | NAME=VALUE]...

A bare NAME turns "#cmakedefine NAME" into "#define NAME". Every other #cmakedefine
becomes "/* #undef NAME */", as in CMake. NAME=VALUE replaces "${NAME}" with VALUE, and
a "${NAME}" with no value is an error.
"""

import re
import sys

template, output, *args = sys.argv[1:]
defined = {a for a in args if "=" not in a}
values = dict(a.split("=", 1) for a in args if "=" in a)

with open(template) as f:
    text = f.read()
text = re.sub(
    r"^#cmakedefine (\w+)$",
    lambda m: f"#define {m[1]}" if m[1] in defined else f"/* #undef {m[1]} */",
    text,
    flags=re.MULTILINE,
)
text = re.sub(r"\$\{(\w+)\}", lambda m: values[m[1]], text)
with open(output, "w", newline="\n") as f:
    f.write(text)
