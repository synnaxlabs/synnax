#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Sets the modification time of each tracked file and directory under a path from its
Git object ID.

go test reuses a cached result only while each file the test read in its module keeps
its size, mode, and modification time. A checkout gives every file a new time, so a test
that reads a fixture never reuses a result from another run. A time from the object ID
changes only when the content does. The working tree must match HEAD.

Usage: pin_mtimes.py <path>
"""

import os
import subprocess
import sys

# 2000-01-01T00:00:00Z. Every time falls in the 20 years after it, so all are in the
# past: go test caches no result that read a file changed in the last 2 seconds.
START_NS = 946_684_800 * 10**9
SPAN_NS = 20 * 365 * 86_400 * 10**9

SYMLINK = "120000"
SUBMODULE = "commit"


def mtime_ns(object_id: str) -> int:
    """Returns the modification time, in nanoseconds since the epoch, for the object."""
    return START_NS + int(object_id, 16) % SPAN_NS


def pin(path: str) -> int:
    """Sets the time of path and of each tracked entry under it, relative to the current
    directory, and returns how many it set. A symbolic link gets its own time, not its
    target's."""
    listing = subprocess.run(
        ["git", "ls-tree", "-r", "-t", "-z", "HEAD", "--", path],
        check=True,
        capture_output=True,
        text=True,
    ).stdout
    count = 0
    for entry in filter(None, listing.split("\0")):
        meta, name = entry.split("\t", 1)
        mode, kind, object_id = meta.split(" ")
        if kind == SUBMODULE:
            continue
        mtime = mtime_ns(object_id)
        os.utime(name, ns=(mtime, mtime), follow_symlinks=mode != SYMLINK)
        count += 1
    return count


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit("usage: pin_mtimes.py <path>")
    path = sys.argv[1]
    print(f"pinned {pin(path)} modification times under {path}")


if __name__ == "__main__":
    main()
