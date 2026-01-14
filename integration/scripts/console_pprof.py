#!/usr/bin/env python3
#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""CPU profiler for the Synnax Console.

Similar to `go tool pprof`, this script collects CPU profiles, heap snapshots,
and Playwright traces from the Console application.

Prerequisites:
    The Console must be running:
        pnpm dev:console

Usage:
    # Launch browser, profile until Enter is pressed
    uv run console-pprof

    # Profile for 30 seconds
    uv run console-pprof --seconds 30

    # Also take a heap snapshot
    uv run console-pprof --heap

    # Capture a Playwright trace (can combine with other options)
    uv run console-pprof --trace

    # Custom output path
    uv run console-pprof --output my_profile.cpuprofile
"""

from __future__ import annotations

import argparse
import shutil
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

try:
    from playwright.sync_api import sync_playwright, Page
except ImportError:
    print("Error: playwright is required. Install with: pip install playwright")
    sys.exit(1)

from console.profiling.writer import ProfileWriter


def generate_filename(profile_type: str) -> str:
    """Generate a timestamped filename for the profile."""
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    return f"console_{timestamp}" if profile_type == "cpu" else f"console_{timestamp}"


def collect_cpu_profile(
    cdp_session, duration_seconds: int | None, writer: ProfileWriter, name: str
) -> Path:
    """Collect a CPU profile for the specified duration.

    Uses the same CDP commands and output format as integration test profiling,
    so profiles can be viewed with speedscope.

    :param cdp_session: CDP session connected to the browser.
    :param duration_seconds: How long to profile in seconds, or None for unlimited.
    :param writer: ProfileWriter instance for saving the profile.
    :param name: Name for the output file (without extension).
    :returns: Path to the saved profile.
    """
    if duration_seconds is not None:
        print(f"Profiling for {duration_seconds} seconds...")
    else:
        print("Profiling until Enter is pressed...")
    print("Interact with the Console now to capture performance data.")
    print()

    # Start profiling (same CDP commands as integration tests)
    cdp_session.send("Profiler.enable")
    cdp_session.send("Profiler.start")

    if duration_seconds is not None:
        # Show countdown
        for remaining in range(duration_seconds, 0, -1):
            print(f"\r  {remaining}s remaining", end="", flush=True)
            time.sleep(1)
        print("\r  Done!            ")
    else:
        # Wait for Enter key (avoids Ctrl+C which kills Playwright connection)
        print("  Press Enter to stop profiling...")
        input()
        print("  Stopping...")

    result = cdp_session.send("Profiler.stop")
    profile = result.get("profile", {})

    path = writer.write_cpu_profile(name, profile)
    print(f"\nProfile saved to: {path}")
    return path


def collect_heap_snapshot(
    cdp_session, page: Page, writer: ProfileWriter, name: str
) -> Path:
    """Take a heap snapshot.

    Uses the same CDP commands and output format as integration test profiling.

    :param cdp_session: CDP session connected to the browser.
    :param page: Playwright page (for timeout).
    :param writer: ProfileWriter instance for saving the snapshot.
    :param name: Name for the output file (without extension).
    :returns: Path to the saved snapshot.
    """
    print("Taking heap snapshot...")

    chunks: list[str] = []

    def on_chunk(params: dict) -> None:
        chunk = params.get("chunk", "")
        if isinstance(chunk, str):
            chunks.append(chunk)

    # Same CDP commands as integration tests
    cdp_session.send("HeapProfiler.enable")
    cdp_session.on("HeapProfiler.addHeapSnapshotChunk", on_chunk)
    cdp_session.send("HeapProfiler.takeHeapSnapshot", {"reportProgress": False})

    # Wait for chunks (same as integration tests)
    page.wait_for_timeout(1000)

    # Use shared ProfileWriter (same output format as integration tests)
    path = writer.write_heap_snapshot(name, chunks)
    size_mb = sum(len(c) for c in chunks) / 1024 / 1024
    print(f"Heap snapshot saved to: {path} ({size_mb:.1f} MB)")
    return path


def open_in_speedscope(profile_path: Path) -> bool:
    """Open the profile in speedscope.

    :param profile_path: Path to the profile file.
    :returns: True if opened successfully.
    """
    if shutil.which("speedscope"):
        print("Opening in speedscope...")
        subprocess.Popen(["speedscope", str(profile_path)])
        return True

    abs_path = profile_path.resolve()
    print()
    print("To view the profile:")
    print(f"  speedscope {abs_path}")
    print()
    print("Or upload to: https://www.speedscope.app/")
    print()
    print("Install speedscope with: npm install -g speedscope")
    return False


def open_heap_in_chrome(snapshot_path: Path) -> None:
    """Print instructions for viewing heap snapshot in Chrome."""
    print()
    print("To view the heap snapshot:")
    print("  1. Open Chrome DevTools (F12)")
    print("  2. Go to Memory tab")
    print("  3. Click 'Load' and select:")
    print(f"     {snapshot_path.resolve()}")


def open_trace_viewer(trace_path: Path) -> None:
    """Print instructions for viewing Playwright trace."""
    abs_path = trace_path.resolve()
    print()
    print("To view the trace:")
    print(f"  npx playwright show-trace {abs_path}")
    print()
    print("Or upload to: https://trace.playwright.dev/")


def wait_for_console_ready(page: Page, timeout: int = 30000) -> bool:
    """Wait for the Console to be ready."""
    try:
        page.wait_for_selector(
            ".pluto-field__username, text=Get Started",
            timeout=timeout
        )
        return True
    except Exception:
        return False


def login_if_needed(page: Page) -> None:
    """Log in to the Console if the login form is present."""
    try:
        username_field = page.locator(".pluto-field__username input").first
        if username_field.is_visible(timeout=2000):
            print("Logging in...")
            username_field.fill("synnax")
            page.locator(".pluto-field__password input").first.fill("seldon")
            page.get_by_role("button", name="Log In").click()
            page.wait_for_selector("text=Get Started", timeout=10000)
            print("Logged in successfully")
    except Exception:
        pass


def find_console_url(page: Page) -> str | None:
    """Try to find a running Console instance.

    Tries port 9090 (embedded console) first, then 5173 (Vite dev server).
    """
    urls_to_try = [
        ("http://localhost:9090", "embedded console"),
        ("http://localhost:5173", "Vite dev server"),
    ]

    for url, description in urls_to_try:
        try:
            page.goto(url, timeout=5000)
            if "Core built without embedded console" in page.content():
                print(f"  {url} - Core running without embedded console, skipping...")
                continue
            print(f"  {url} - Found Console ({description})")
            return url
        except Exception:
            print(f"  {url} - Not available")
            continue

    return None


def run_profiler(args: argparse.Namespace) -> None:
    """Run the profiler with the given arguments."""
    # Set up output directory and writer (shared with integration tests)
    if args.output:
        output_dir = Path(args.output).parent.resolve()
        name = Path(args.output).stem
    else:
        output_dir = Path.cwd() / "profiles"
        name = generate_filename("heap" if args.heap else "cpu")

    writer = ProfileWriter(output_dir)

    print("Launching browser...")
    print()

    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=False,
            args=["--start-maximized"],
        )

        context = browser.new_context(
            no_viewport=True,
        )

        # Start tracing BEFORE page creation for complete capture (same as client)
        if args.trace:
            context.tracing.start(screenshots=True, snapshots=True, sources=True)

        page = context.new_page()

        # Find Console URL (try 9090 first, then 5173)
        print("Looking for Console...")
        url = find_console_url(page)
        if url is None:
            print()
            print("Error: Could not find a running Console")
            print()
            print("Start the Console with one of:")
            print("  pnpm dev:console        # Tauri app (uses Vite at 5173)")
            print("  pnpm dev:console-vite   # Vite only (port 5173)")
            print()
            print("Or run Core with embedded console:")
            print("  go run main.go start -tags console")
            browser.close()
            sys.exit(1)

        if not wait_for_console_ready(page):
            print("Warning: Console may not be fully loaded")

        login_if_needed(page)

        # Create CDP session (same approach as integration tests)
        cdp_session = context.new_cdp_session(page)

        try:
            print("Console loaded. Starting profiler...")
            print()

            # CPU profile is always collected
            cpu_path = collect_cpu_profile(cdp_session, args.seconds, writer, name)

            heap_path = None
            if args.heap:
                heap_path = collect_heap_snapshot(cdp_session, page, writer, name)

            trace_path = None
            if args.trace:
                trace_path = output_dir / f"{name}.trace.zip"
                context.tracing.stop(path=str(trace_path))
                print(f"\nTrace saved to: {trace_path}")

            # Show viewing instructions last
            if heap_path:
                open_heap_in_chrome(heap_path)
            if trace_path:
                open_trace_viewer(trace_path)
            open_in_speedscope(cpu_path)

        finally:
            try:
                cdp_session.detach()
            except Exception:
                pass
            try:
                context.close()
            except Exception:
                pass
            try:
                browser.close()
            except Exception:
                pass


def main() -> None:
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="CPU profiler for the Synnax Console (similar to go tool pprof)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                      # CPU profile until Enter is pressed
  %(prog)s --seconds 30         # CPU profile for 30 seconds
  %(prog)s --heap               # CPU profile + heap snapshot
  %(prog)s --trace              # CPU profile + Playwright trace
  %(prog)s --trace --heap       # CPU profile + trace + heap snapshot

Prerequisites:
  Start the Console:
    pnpm dev:console

View profiles with speedscope:
  npm install -g speedscope
  speedscope <profile.cpuprofile>
        """,
    )
    parser.add_argument(
        "--seconds",
        "-s",
        type=int,
        default=None,
        help="Duration to profile in seconds (default: until Enter)",
    )
    parser.add_argument(
        "--heap",
        action="store_true",
        help="Also take a heap snapshot (in addition to CPU profile)",
    )
    parser.add_argument(
        "--trace",
        "-t",
        action="store_true",
        help="Capture a Playwright trace (can combine with --heap or CPU profiling)",
    )
    parser.add_argument(
        "--output",
        "-o",
        type=Path,
        help="Output file path (default: auto-generated in profiles/)",
    )
    args = parser.parse_args()

    run_profiler(args)


if __name__ == "__main__":
    main()
