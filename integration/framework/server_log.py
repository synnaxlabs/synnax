#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Server-log capture helpers.

The Synnax server's stdout/stderr is redirected to a log file by the
``start_core`` scripts (``~/synnax-data/synnax-core.log`` by default; overridable
via ``SYNNAX_SERVER_LOG``). The conductor doesn't own that lifecycle. It just
reads the file at run-boundary times to compose the bundle.

Per-test slicing uses byte offsets rather than timestamp parsing so that it
works regardless of whether the server is configured for text or JSON logging.
"""

import os
from pathlib import Path

DEFAULT_SERVER_LOG = Path.home() / "synnax-data" / "synnax-core.log"


def server_log_path() -> Path | None:
    """Return the configured server log path if it exists, otherwise None."""
    p = Path(os.environ.get("SYNNAX_SERVER_LOG", str(DEFAULT_SERVER_LOG)))
    return p if p.exists() else None


def current_offset() -> int | None:
    """Return the current size in bytes of the server log, or None if missing."""
    p = server_log_path()
    if p is None:
        return None
    try:
        return p.stat().st_size
    except OSError:
        return None


def slice_to(dst: Path, start: int | None, end: int | None) -> bool:
    """Copy server-log bytes ``[start, end)`` into ``dst``.

    Returns True if any bytes were written. False if the source is missing,
    the range is empty, or any I/O error occurs (server log capture must
    never break test execution).
    """
    src = server_log_path()
    if src is None or start is None or end is None or end <= start:
        return False
    try:
        dst.parent.mkdir(parents=True, exist_ok=True)
        with open(src, "rb") as fsrc:
            fsrc.seek(start)
            chunk = fsrc.read(end - start)
        with open(dst, "wb") as fdst:
            fdst.write(chunk)
        return True
    except OSError:
        return False


def copy_range(dst: Path, start: int | None) -> bool:
    """Copy the server log from ``start`` to current EOF into ``dst``.

    Used at conductor-end to capture this run's slice of the (possibly long-lived)
    server log into the run bundle. Passes ``start`` through to ``slice_to``, so
    a ``None`` start (the server log wasn't measurable at conductor init) skips
    the capture rather than dumping the whole file, which would mix in content
    from previous runs.
    """
    src = server_log_path()
    if src is None:
        return False
    try:
        end = src.stat().st_size
    except OSError:
        return False
    return slice_to(dst, start, end)
