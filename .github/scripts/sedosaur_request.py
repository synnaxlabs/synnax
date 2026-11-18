#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reads a comment for a deliberate request for a Sedosaur review.

    BODY=<comment> python3 sedosaur_request.py

Prints `requested=` and `repeat=` lines for GITHUB_OUTPUT. A request is a line that
starts with `@sedosaur review`, outside a quote, a code span, and a code fence. It asks
for a repeat when `repeat=true` is one of its words.
"""

import os
import re
from dataclasses import dataclass

CODE_SPAN = re.compile(r"`[^`]*`")


@dataclass(frozen=True)
class Request:
    requested: bool
    repeat: bool


def parse(body: str) -> Request:
    requested = repeat = False
    fenced = False
    for line in body.splitlines():
        line = line.strip()
        if line.startswith(("```", "~~~")):
            fenced = not fenced
            continue
        if fenced or line.startswith(">"):
            continue
        words = CODE_SPAN.sub(" ", line).lower().split()
        if words[:2] != ["@sedosaur", "review"]:
            continue
        requested = True
        repeat = repeat or "repeat=true" in words[2:]
    return Request(requested=requested, repeat=repeat)


if __name__ == "__main__":
    request = parse(os.environ.get("BODY", ""))
    print(f"requested={str(request.requested).lower()}")
    print(f"repeat={str(request.repeat).lower()}")
