#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Run-directory creation.

Every conductor invocation creates a single run directory under
``integration/tests/results/run-<ts>[-name]/``. Inside it:

- ``server.log``      -> server stdout captured from start_core
- ``summary.json``    -> structured per-test outcomes
- ``tests/<name>/``   -> per-test bundle (trace.zip, sliced server.log,
                         screenshots, exports)

``ExecutionClient`` creates each test's bundle and passes it to the test case as
``results_dir``.
"""

from datetime import UTC, datetime
from pathlib import Path

_RESULTS_ROOT = Path(__file__).resolve().parent.parent / "tests" / "results"


def create(name: str | None = None) -> Path:
    """Create and return the bundle directory for this conductor invocation.

    The path is ``_RESULTS_ROOT/run-<UTC-timestamp>[-<name>]/``. The
    ``_RESULTS_ROOT/latest`` symlink is updated to point at it on filesystems
    that support symlinks.
    """
    ts = datetime.now(UTC).strftime("%Y%m%dT%H%M%SZ")
    slug = f"run-{ts}" if not name else f"run-{ts}-{name}"
    run_dir = _RESULTS_ROOT / slug
    run_dir.mkdir(parents=True, exist_ok=True)
    (run_dir / "tests").mkdir(exist_ok=True)
    _update_latest_symlink(run_dir)
    return run_dir


def _update_latest_symlink(target: Path) -> None:
    link = _RESULTS_ROOT / "latest"
    try:
        if link.is_symlink() or link.exists():
            link.unlink()
        link.symlink_to(target.name)
    except OSError:
        # Windows without dev-mode and some restricted filesystems disallow
        # symlinks. The bundle is still usable; just no `latest` shortcut.
        pass
