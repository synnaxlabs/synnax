#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import gc
import platform
import re
import subprocess
import sys


# Also suppress stderr for WebSocket errors
class WebSocketErrorFilter:
    def __init__(self):
        self.original_stderr = sys.stderr

    def write(self, text):
        if any(
            phrase in text
            for phrase in [
                "keepalive ping",
                "1011",
                "timed out while closing connection",
                "ConnectionClosedError",
                "WebSocketException",
            ]
        ):
            return
        self.original_stderr.write(text)

    def flush(self):
        self.original_stderr.flush()


# More aggressive WebSocket error suppression
def ignore_websocket_errors(type, value, traceback):
    error_str = str(value)
    if any(
        phrase in error_str
        for phrase in [
            "keepalive ping",
            "1011",
            "timed out while closing connection",
            "ConnectionClosedError",
            "WebSocketException",
        ]
    ):
        return
    sys.__excepthook__(type, value, traceback)


def validate_and_sanitize_name(name: str) -> str:
    """Sanitize name to contain only alphanumeric characters, hyphens, and underscores."""
    sanitized = re.sub(r"[^a-zA-Z0-9_-]", "", name)

    if not sanitized:
        raise ValueError("Name must contain at least one alphanumeric character")

    sanitized = sanitized.strip("_-")
    if not sanitized:
        raise ValueError("Name cannot consist only of hyphens and underscores")

    return sanitized


def get_machine_info():
    """Get machine information programmatically."""

    # TODO: SY-2811 Move Test Framework Utils to X/os

    system = platform.system()

    if system == "Darwin":  # macOS
        try:
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
                return "macOS"
        except:
            return "macOS"

    elif system == "Linux":
        try:
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
                return "Linux"
        except:
            return "Linux"

    elif system == "Windows":
        try:
            # Get Windows version info
            result = subprocess.run(
                ["wmic", "os", "get", "Caption"],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode == 0:
                lines = result.stdout.strip().split("\n")
                if len(lines) > 1:
                    return lines[1].strip()
                else:
                    return "Windows"
            else:
                return "Windows"
        except:
            return "Windows"

    else:
        return system


def get_memory_info():
    """Get memory information."""
    try:
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
            result = subprocess.run(
                ["wmic", "computersystem", "get", "TotalPhysicalMemory"],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode == 0:
                lines = result.stdout.strip().split("\n")
                if len(lines) > 1:
                    mem_bytes = int(lines[1].strip())
                    mem_gb = mem_bytes // (1024**3)
                    return f"{mem_gb}GB RAM"
    except:
        pass

    return ""


def get_synnax_version():
    """Get the current Synnax version from the VERSION file."""
    try:
        # Try to read from the VERSION file in the synnax package
        version_file = "../../../core/pkg/version/VERSION"
        with open(version_file, "r") as f:
            version = f.read().strip()
            return version
    except:
        try:
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
                if version.startswith("v"):
                    version = version[1:]
                return version
        except:
            pass

    return "unknown"
