#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass


def split_csv(value: str) -> list[str]:
    """Split a comma-separated string into lowercased, stripped tokens."""
    return [s.strip().lower() for s in value.split(",") if s.strip()]


def _any_match(patterns: list[str] | None, name: str) -> bool:
    """True if any pattern is a substring of name (case-insensitive)."""
    if patterns is None:
        return False
    lower = name.lower()
    return any(pat in lower for pat in patterns)


@dataclass
class TargetFilter:
    """Flexible test filter supporting substring matching at every level."""

    file_filter: list[str] | None = None
    sequence_filter: str | None = None
    case_filter: list[str] | None = None
    exclude: list[str] | None = None

    @property
    def is_empty(self) -> bool:
        return (
            not self.file_filter
            and self.sequence_filter is None
            and self.case_filter is None
            and self.exclude is None
        )

    def matches_sequence(self, seq_name: str) -> bool:
        if self.sequence_filter is None:
            return True
        return self.sequence_filter.lower() in seq_name.lower()

    def matches_case(self, case_path: str) -> bool:
        if self.case_filter is None:
            return True
        return _any_match(self.case_filter, case_path)

    def excluded(self, name: str) -> bool:
        return _any_match(self.exclude, name)


def parse_target(target: str) -> TargetFilter:
    """Parse a target path into a TargetFilter.

    Supported formats:
        "console"                  -> file=console
        "console/..."              -> file=console
        "driver/modbus"            -> file=driver, case_filter="modbus"
        "console/lifecycle/..."    -> file=console, sequence_filter="lifecycle"
        "console/lifecycle/label"  -> file=console, sequence_filter="lifecycle",
                                      case_filter="label"

    2-part paths treat the second segment as a case_filter (substring),
    so "driver/modbus" matches cases like "driver/modbus_read".

    3-part paths use the second segment as a sequence_filter and the third
    as a case_filter.

    "..." at any position is treated as a wildcard (no filter).
    """
    path = target.strip().lstrip("/")
    if not path:
        raise ValueError(f"Target path cannot be empty: {target!r}")

    parts = [p for p in path.split("/") if p]
    if not parts:
        raise ValueError(f"Target path cannot be empty: {target!r}")

    file_filter = [f.strip() for f in parts[0].split(",") if f.strip()]
    if not file_filter:
        raise ValueError(f"Target path cannot be empty: {target!r}")

    sequence_filter: str | None = None
    case_filter: list[str] | None = None

    if len(parts) == 2:
        if parts[1] != "...":
            case_filter = split_csv(parts[1])
    elif len(parts) >= 3:
        if parts[1] != "...":
            sequence_filter = parts[1]
        if parts[2] != "...":
            case_filter = split_csv(parts[2])

    return TargetFilter(
        file_filter=file_filter,
        sequence_filter=sequence_filter,
        case_filter=case_filter,
    )
