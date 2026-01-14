#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""CDP-based profiler implementation for Playwright browser sessions."""

from playwright.sync_api import BrowserContext, CDPSession, Page

from console.profiling.config import ProfilerConfig
from console.profiling.writer import ProfileWriter


class CDPProfiler:
    """Profiler that uses Chrome DevTools Protocol for collecting performance data.

    This profiler integrates with Playwright browser sessions to collect:
    - CPU profiles via Profiler.start/stop
    - Heap snapshots via HeapProfiler.takeHeapSnapshot
    - Playwright traces via context.tracing

    Note: Tracing should be started on the context BEFORE page creation for complete
    capture. Use start_tracing() before creating the page, then start_cdp_profiling()
    after the page is ready.

    :param page: Playwright page to profile.
    :param context: Playwright browser context for tracing.
    :param config: Profiler configuration.
    """

    def __init__(
        self,
        page: Page,
        context: BrowserContext,
        config: ProfilerConfig,
    ) -> None:
        self._page = page
        self._context = context
        self._config = config
        self._writer = ProfileWriter(config.output_dir)
        self._cdp_session: CDPSession | None = None
        self._heap_chunks: list[str] = []

    @property
    def config(self) -> ProfilerConfig:
        """Get the profiler configuration."""
        return self._config

    def start(self) -> None:
        """Start all enabled profiling features.

        Note: For complete tracing, call start_tracing() on the context before
        creating the page, then call start_cdp_profiling() after page creation.
        This method is a convenience that starts both, but tracing may miss
        initial page load.
        """
        if self._config.tracing:
            self._start_tracing()

        self.start_cdp_profiling()

    def start_cdp_profiling(self) -> None:
        """Start CDP-based profiling features.

        Call this after the page is ready. Initializes CDP session and starts
        CPU profiling as configured.
        """
        if not self._config.requires_cdp:
            return

        self._init_cdp_session()

        if self._cdp_session is not None:
            if self._config.cpu_profiling:
                self._start_cpu_profiling()

    def stop(self, test_name: str) -> None:
        """Stop all profiling and save results.

        :param test_name: Name of the test, used for output file names.
        """
        if self._cdp_session is not None:
            if self._config.heap_snapshot:
                self._collect_heap_snapshot(test_name)
            if self._config.cpu_profiling:
                self._stop_cpu_profiling(test_name)

        if self._config.tracing:
            self._stop_tracing(test_name)

    def close(self) -> None:
        """Clean up CDP session."""
        if self._cdp_session is not None:
            try:
                self._cdp_session.detach()
            except Exception:
                pass
            self._cdp_session = None

    def _init_cdp_session(self) -> None:
        """Initialize CDP session for profiling. Only works with Chromium."""
        try:
            self._cdp_session = self._context.new_cdp_session(self._page)
        except Exception:
            self._cdp_session = None

    def _start_tracing(self) -> None:
        """Start Playwright tracing on the browser context."""
        self._context.tracing.start(screenshots=True, snapshots=True, sources=True)

    def _stop_tracing(self, test_name: str) -> None:
        """Stop Playwright tracing and save trace to disk."""
        trace_path = self._config.output_dir / f"{test_name}.trace.zip"
        self._context.tracing.stop(path=str(trace_path))

    def _start_cpu_profiling(self) -> None:
        """Start CPU profiling via CDP."""
        if self._cdp_session is None:
            return
        self._cdp_session.send("Profiler.enable")
        self._cdp_session.send("Profiler.start")

    def _stop_cpu_profiling(self, test_name: str) -> None:
        """Stop CPU profiling and save profile to disk."""
        if self._cdp_session is None:
            return
        result = self._cdp_session.send("Profiler.stop")
        profile = result.get("profile", {})
        self._writer.write_cpu_profile(test_name, profile)

    def _collect_heap_snapshot(self, test_name: str) -> None:
        """Take a heap snapshot and save to disk."""
        if self._cdp_session is None:
            return
        self._heap_chunks = []

        def on_chunk(params: dict[str, object]) -> None:
            chunk = params.get("chunk", "")
            if isinstance(chunk, str):
                self._heap_chunks.append(chunk)

        # HeapProfiler.enable is required before takeHeapSnapshot
        self._cdp_session.send("HeapProfiler.enable")
        self._cdp_session.on("HeapProfiler.addHeapSnapshotChunk", on_chunk)
        self._cdp_session.send(
            "HeapProfiler.takeHeapSnapshot", {"reportProgress": False}
        )

        # Give time for chunks to be collected
        self._page.wait_for_timeout(1000)

        self._writer.write_heap_snapshot(test_name, self._heap_chunks)
