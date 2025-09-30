#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import platform
import re
import subprocess
import sys
from typing import Any

# SY-2920: Websocket Error handling improvements
WEBSOCKET_ERROR_PATTERNS = [
    "1011",
    "keepalive ping timeout",
    "keepalive ping failed",
    "keepalive ping",
    "timed out while closing connection",
    "ConnectionClosedError",
    "WebSocketException",
]


# Also suppress stderr for WebSocket errors
class WebSocketErrorFilter:
    def __init__(self) -> None:
        self.original_stderr = sys.stderr

    def write(self, text: str) -> None:
        if any(phrase in text for phrase in WEBSOCKET_ERROR_PATTERNS):
            return
        self.original_stderr.write(text)

    def flush(self) -> None:
        self.original_stderr.flush()


# More aggressive WebSocket error suppression
def ignore_websocket_errors(
    type: type[BaseException], value: BaseException, traceback: Any
) -> None:
    error_str = str(value)
    if any(phrase in error_str for phrase in WEBSOCKET_ERROR_PATTERNS):
        return
    sys.__excepthook__(type, value, traceback)


def is_websocket_error(error: Exception) -> bool:
    """Check if an exception is a WebSocket-related error that should be ignored."""
    error_str = str(error)
    return any(phrase in error_str for phrase in WEBSOCKET_ERROR_PATTERNS)


def is_ci() -> bool:
    """Check if running in a CI environment."""
    return any(
        env_var in os.environ
        for env_var in ["CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"]
    )


def validate_and_sanitize_name(name: str) -> str:
    """Sanitize name to contain only alphanumeric characters, hyphens, and underscores."""
    sanitized = re.sub(r"[^a-zA-Z0-9_-]", "", name)

    if not sanitized:
        raise ValueError("Name must contain at least one alphanumeric character")

    sanitized = sanitized.strip("_-")
    if not sanitized:
        raise ValueError("Name cannot consist only of hyphens and underscores")

    return sanitized


def get_machine_info() -> str:
    """Get machine information programmatically."""

    # TODO: SY-2811 Move Test Framework Utils to X/os

    system = platform.system()

    if system == "Darwin":  # macOS
        # Try to get Apple Silicon info
        result = subprocess.run(
            ["sysctl", "-n", "machdep.cpu.brand_string"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0:
            cpu_info = result.stdout.strip()
            if "Apple" in cpu_info:
                # Extract M1/M2/M3 info
                if "M1" in cpu_info:
                    return "Apple Silicon M1"
                elif "M2" in cpu_info:
                    return "Apple Silicon M2"
                elif "M3" in cpu_info:
                    return "Apple Silicon M3"
                elif "M4" in cpu_info:
                    return "Apple Silicon M4"
                elif "M5" in cpu_info:
                    return "Apple Silicon M5"
                else:
                    return "Apple Silicon Mac"
            else:
                return "Intel Mac"
        else:
            raise RuntimeError(
                f"Failed to get macOS CPU information: sysctl command returned {result.returncode}"
            )

    elif system == "Linux":
        # Try to get distribution info
        result = subprocess.run(
            ["lsb_release", "-d"], capture_output=True, text=True, timeout=5
        )
        if result.returncode == 0:
            distro = result.stdout.split("\t")[1].strip()
            return distro
        else:
            # Try reading from /etc/os-release
            with open("/etc/os-release", "r") as f:
                for line in f:
                    if line.startswith("PRETTY_NAME="):
                        distro = line.split("=")[1].strip().strip('"')
                        return distro
            raise RuntimeError(
                "Unable to determine Linux distribution from lsb_release or /etc/os-release"
            )

    elif system == "Windows":
        # Get Windows version info
        result = subprocess.run(
            [
                "powershell",
                "-Command",
                "(Get-CimInstance Win32_OperatingSystem).Caption",
            ],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0:
            return result.stdout.strip()
        else:
            raise RuntimeError(
                f"Failed to get Windows version: PowerShell command returned {result.returncode}"
            )

    else:
        return system


def get_memory_info() -> str:
    """Get memory information."""
    if platform.system() == "Darwin":  # macOS
        result = subprocess.run(
            ["sysctl", "-n", "hw.memsize"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0:
            mem_bytes = int(result.stdout.strip())
            mem_gb = mem_bytes // (1024**3)
            return f"{mem_gb}GB RAM"
    elif platform.system() == "Linux":
        with open("/proc/meminfo", "r") as f:
            for line in f:
                if line.startswith("MemTotal:"):
                    mem_kb = int(line.split()[1])
                    mem_gb = mem_kb // (1024**2)
                    return f"{mem_gb}GB RAM"
    elif platform.system() == "Windows":
        # Use PowerShell instead of wmic (deprecated in Win11)
        result = subprocess.run(
            [
                "powershell",
                "-Command",
                "(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory",
            ],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0:
            mem_bytes = int(result.stdout.strip())
            mem_gb = mem_bytes // (1024**3)
            return f"{mem_gb}GB RAM"

    raise RuntimeError(f"Unable to get memory information for {platform.system()}")


def get_synnax_version() -> str:
    """Get the current Synnax version from the VERSION file."""

    # SY-2917

    # Try multiple possible paths for the VERSION file based on working directory
    possible_paths = [
        "../core/pkg/version/VERSION",  # From integration directory (CI environment)
        "core/pkg/version/VERSION",  # From synnax root directory
        "../../../core/pkg/version/VERSION",  # Original deep nested path
    ]

    for version_file in possible_paths:
        if os.path.exists(version_file):
            try:
                with open(version_file, "r") as f:
                    version = f.read().strip()
                    if version:  # Make sure it's not empty
                        return version
            except (FileNotFoundError, PermissionError):
                continue  # Try next path

    # Fallback: try to get version from git tags
    result = subprocess.run(
        ["git", "describe", "--tags", "--abbrev=0"],
        capture_output=True,
        text=True,
        timeout=5,
    )
    if result.returncode == 0:
        version = result.stdout.strip()
        # Remove 'v' prefix if present
        if version and version.startswith("v"):
            version = version[1:]
        if version:  # Make sure it's not empty
            return version

    raise RuntimeError(
        "Unable to determine Synnax version from VERSION file or git tags"
    )
