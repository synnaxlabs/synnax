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
            else:
                return "Intel Mac"
        else:
            raise RuntimeError(
                f"Failed to get macOS CPU information: "
                f"sysctl command returned {result.returncode}"
            )

    elif system == "Linux":
        result = subprocess.run(
            ["lsb_release", "-d"], capture_output=True, text=True, timeout=5
        )
        if result.returncode == 0:
            return result.stdout.split("\t")[1].strip()
        else:
            with open("/etc/os-release", "r") as f:
                for line in f:
                    if line.startswith("PRETTY_NAME="):
                        return line.split("=")[1].strip().strip('"')
            raise RuntimeError(
                "Unable to determine Linux distribution "
                "from lsb_release or /etc/os-release"
            )

    elif system == "Windows":
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
                f"Failed to get Windows version: "
                f"PowerShell command returned {result.returncode}"
            )

    else:
        return system


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
            return f"{mem_bytes // (1024 ** 3)}GB RAM"

    elif system == "Linux":
        with open("/proc/meminfo", "r") as f:
            for line in f:
                if line.startswith("MemTotal:"):
                    mem_kb = int(line.split()[1])
                    return f"{mem_kb // (1024 ** 2)}GB RAM"

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
            return f"{mem_bytes // (1024 ** 3)}GB RAM"

    raise RuntimeError(f"Unable to get memory information for {system}")


def get_cpu_cores() -> int:
    """Get the number of CPU cores."""
    return multiprocessing.cpu_count()


def get_platform() -> Platform:
    """Get the current platform as a lowercase string."""
    system = platform.system()
    if system == "Linux":
        return "linux"
    elif system == "Darwin":
        return "macos"
    elif system == "Windows":
        return "windows"
    raise RuntimeError(f"Unsupported platform: {system}")
