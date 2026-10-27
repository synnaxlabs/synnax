#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import multiprocessing
import platform
import subprocess
from contextlib import suppress
from pathlib import Path
from typing import Literal

Platform = Literal["linux", "macos", "windows"]


def get_machine_info() -> str:
    """Get machine information programmatically."""
    system = platform.system()

    if system == "Darwin":
        result = subprocess.run(
            ["sysctl", "-n", "machdep.cpu.brand_string"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0:
            cpu_info = result.stdout.strip()
            if "Apple" in cpu_info:
                for chip in ("M1", "M2", "M3", "M4", "M5"):
                    if chip in cpu_info:
                        return f"Apple Silicon {chip}"
                return "Apple Silicon Mac"
            return "Intel Mac"
        raise RuntimeError(
            f"Failed to get macOS CPU information: "
            f"sysctl command returned {result.returncode}"
        )

    if system == "Linux":
        return _get_linux_distribution()

    if system == "Windows":
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
        raise RuntimeError(
            f"Failed to get Windows version: "
            f"PowerShell command returned {result.returncode}"
        )

    return system


def _get_linux_distribution(
    lsb_release: str = "lsb_release", os_release: Path = Path("/etc/os-release")
) -> str:
    """Returns the Linux distribution name. Reads os_release when lsb_release fails
    or is not installed.

    :param lsb_release: The lsb_release command to run.
    :param os_release: The os-release file to read on fallback.
    :raises OSError: If lsb_release fails and os_release cannot be read.
    :raises RuntimeError: If lsb_release fails and os_release has no PRETTY_NAME.
    """
    with suppress(FileNotFoundError):
        result = subprocess.run(
            [lsb_release, "-d"], capture_output=True, text=True, timeout=5
        )
        if result.returncode == 0:
            return result.stdout.split("\t")[1].strip()
    with os_release.open() as f:
        for line in f:
            if line.startswith("PRETTY_NAME="):
                return line.split("=")[1].strip().strip('"')
    raise RuntimeError(
        f"Unable to determine Linux distribution from {lsb_release} or {os_release}"
    )


def get_memory_info() -> str:
    """Get memory information."""
    system = platform.system()

    if system == "Darwin":
        result = subprocess.run(
            ["sysctl", "-n", "hw.memsize"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0:
            mem_bytes = int(result.stdout.strip())
            return f"{mem_bytes // (1024**3)}GB RAM"

    elif system == "Linux":
        with open("/proc/meminfo") as f:
            for line in f:
                if line.startswith("MemTotal:"):
                    mem_kb = int(line.split()[1])
                    return f"{mem_kb // (1024**2)}GB RAM"

    elif system == "Windows":
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
            return f"{mem_bytes // (1024**3)}GB RAM"

    raise RuntimeError(f"Unable to get memory information for {system}")


def get_cpu_cores() -> int:
    """Get the number of CPU cores."""
    return multiprocessing.cpu_count()


def get_platform() -> Platform:
    """Get the current platform as a lowercase string."""
    system = platform.system()
    if system == "Linux":
        return "linux"
    if system == "Darwin":
        return "macos"
    if system == "Windows":
        return "windows"
    raise RuntimeError(f"Unsupported platform: {system}")
