#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Run-directory and per-test bundle path management.

Every conductor invocation creates a single run directory under
``integration/tests/results/run-<ts>[-name]/``. Inside it:

- ``server.log``      -> server stdout captured from start_core
- ``summary.json``    -> structured per-test outcomes
- ``tests/<name>/``   -> per-test bundle (trace.zip, sliced server.log,
                         screenshots, exports)

Per-test scoping is thread-local. ``ExecutionClient`` calls ``set_test_dir``
at the start of each test thread; helpers that call ``resolve_results_path``
transparently route writes into the active test's bundle.
"""

import os
import threading
from datetime import datetime, timezone
from pathlib import Path

RESULTS_ROOT = Path(__file__).resolve().parent.parent / "tests" / "results"

_thread_local = threading.local()
_run_dir: Path | None = None


def init_run_dir(name: str | None = None) -> Path:
    """Create and return the bundle directory for this conductor invocation.

    The path is ``RESULTS_ROOT/run-<UTC-timestamp>[-<name>]/``. The
    ``RESULTS_ROOT/latest`` symlink is updated to point at it on filesystems
    that support symlinks.
    """
    global _run_dir
    ts = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    slug = f"run-{ts}" if not name else f"run-{ts}-{name}"
    _run_dir = RESULTS_ROOT / slug
    _run_dir.mkdir(parents=True, exist_ok=True)
    (_run_dir / "tests").mkdir(exist_ok=True)
    _update_latest_symlink(_run_dir)
    return _run_dir


def _update_latest_symlink(target: Path) -> None:
    link = RESULTS_ROOT / "latest"
    try:
        if link.is_symlink() or link.exists():
            link.unlink()
        link.symlink_to(target.name)
    except OSError:
        # Windows without dev-mode and some restricted filesystems disallow
        # symlinks. The bundle is still usable; just no `latest` shortcut.
        pass


def get_run_dir() -> Path | None:
    """Return the active run directory, or None if no run has been initialized."""
    return _run_dir


def get_test_dir() -> Path | None:
    """Return the current thread's active test bundle dir, or None."""
    return getattr(_thread_local, "test_dir", None)


def set_test_dir(path: Path | None) -> None:
    """Set or clear the current thread's active test bundle dir.

    Must be called from inside the test thread, since the dir is thread-local.
    Passing ``None`` clears the binding.
    """
    if path is None:
        if hasattr(_thread_local, "test_dir"):
            del _thread_local.test_dir
        return
    path.mkdir(parents=True, exist_ok=True)
    _thread_local.test_dir = path


def resolve_results_path(filename: str) -> str:
    """Resolve a results filename to an absolute path in the active scope.

    Resolution order: active test dir → run dir → legacy flat results root.
    The chosen directory is created on demand.
    """
    base = get_test_dir() or _run_dir or RESULTS_ROOT
    base.mkdir(parents=True, exist_ok=True)
    return os.path.join(str(base), filename)
