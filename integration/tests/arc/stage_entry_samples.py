#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_indexed_pair, create_virtual_channels
from tests.arc.arc import ArcCase
from x import random_name

START_CMD = "start_entry_samples_cmd"
SEQUENCES = ["drop", "straddle", "milli", "nano"]

SECOND = sy.TimeSpan.SECOND
MILLISECOND = sy.TimeSpan.MILLISECOND
NANOSECOND = sy.TimeSpan.NANOSECOND
# The value every sample taken before entry carries. Seeing it on pt_seen is a leak.
BEFORE_ENTRY = -1.0


def _ch(p: str, name: str, tag: str) -> str:
    return f"{p}_{name}_{tag}"


def _sequence(p: str, tag: str) -> str:
    log, go, entered, pt, seen, probe, probe_seen = (
        _ch(p, n, tag)
        for n in ("log", "go_cmd", "entered", "pt", "pt_seen", "probe", "probe_seen")
    )
    return f"""
sequence {p}_main {{
    stage a {{
        "a" -> {log}
        {go} => next
    }}
    stage b {{
        1 -> {entered}
        "b" -> {log}
        {pt} -> {seen}
        {probe} -> {probe_seen}
    }}
}}
{START_CMD} => {p}_main
"""


class StageEntrySamples(ArcCase):
    """A stage sees only samples taken after it was entered (SY-5456).

    The test stamps every `pt` sample itself; samples taken before entry carry
    BEFORE_ENTRY. `entered` carries the entry cycle stamp. A `probe` seen after a
    pre-entry sample proves the runtime already handled it, so a leak would show.
    """

    arc_name_prefix = "StageEntrySamples"
    start_cmd_channel = START_CMD

    def setup(self) -> None:
        self.tag = random_name()
        self.arc_source = "".join(_sequence(p, self.tag) for p in SEQUENCES)
        self.subscribe_channels = [
            self._ch(p, n) for p in SEQUENCES for n in ("log", "pt_seen", "probe_seen")
        ] + [self._entered_time(p) for p in SEQUENCES]
        for p in SEQUENCES:
            create_indexed_pair(
                self.client, self._ch(p, "pt"), data_type=sy.DataType.FLOAT64
            )
            create_indexed_pair(
                self.client, self._ch(p, "pt_seen"), data_type=sy.DataType.FLOAT64
            )
            create_indexed_pair(
                self.client, self._ch(p, "entered"), data_type=sy.DataType.UINT8
            )
            create_virtual_channels(
                self.client,
                [
                    (self._ch(p, "go_cmd"), sy.DataType.UINT8),
                    (self._ch(p, "probe"), sy.DataType.UINT8),
                    (self._ch(p, "probe_seen"), sy.DataType.UINT8),
                    (self._ch(p, "log"), sy.DataType.STRING),
                ],
            )
        super().setup()

    def _ch(self, p: str, name: str) -> str:
        return _ch(p, name, self.tag)

    def _entered_time(self, p: str) -> str:
        return f"{self._ch(p, 'entered')}_time"

    def _enter_b(self, p: str) -> sy.TimeStamp:
        """Move the sequence into stage B and return its entry cycle stamp."""
        self.wait_for_eq(self._ch(p, "log"), "a")
        self.writer.write(self._ch(p, "go_cmd"), 1)
        self.wait_for_gt(self._entered_time(p), 0)
        return sy.TimeStamp(int(self.read_tlm(self._entered_time(p), 0)))

    def _write_pt(
        self, p: str, stamps: list[sy.TimeStamp], values: list[float]
    ) -> None:
        pt = self._ch(p, "pt")
        with self.client.open_writer(
            start=stamps[0],
            channels=[f"{pt}_time", pt],
            enable_auto_commit=True,
        ) as w:
            w.write({f"{pt}_time": stamps, pt: values})

    def _expect_not_leaked(self, p: str) -> None:
        self.writer.write(self._ch(p, "probe"), 1)
        self.wait_for_eq(self._ch(p, "probe_seen"), 1)
        if self.read_tlm(self._ch(p, "pt_seen")) == BEFORE_ENTRY:
            self.fail(f"a sample taken before entry reached stage B of {p}_main")

    def _verify_before_entry(self, p: str, before: list[sy.TimeSpan]) -> None:
        """Deliver samples stamped `before` earlier than entry; none may reach B."""
        self.log(f"[{p}] Samples stamped {before} before entry must never reach B")
        entered = self._enter_b(p)
        self._write_pt(p, [entered - b for b in before], [BEFORE_ENTRY] * len(before))
        self._expect_not_leaked(p)

    def _verify_straddle(self) -> None:
        self.log("[straddle] Only the samples taken at or after entry reach B")
        entered = self._enter_b("straddle")
        self._write_pt(
            "straddle",
            [
                entered - 5 * SECOND,
                entered - 4 * SECOND,
                entered + 2 * SECOND,
                entered + 3 * SECOND,
            ],
            [BEFORE_ENTRY, BEFORE_ENTRY, 1.0, 2.0],
        )
        seen_ch = self._ch("straddle", "pt_seen")
        self.wait_for_eq(seen_ch, 2.0)
        frame = self.client.read(
            sy.TimeRange(entered - 10 * SECOND, entered + 10 * SECOND), [seen_ch]
        )
        seen = frame[seen_ch].to_numpy().tolist()
        if seen != [1.0, 2.0]:
            self.fail(f"{seen_ch} holds {seen}, expected [1.0, 2.0]")

    def verify_sequence_execution(self) -> None:
        self._verify_before_entry("drop", [6 * SECOND, 5 * SECOND, 4 * SECOND])
        self._verify_straddle()
        self._verify_before_entry("milli", [MILLISECOND])
        self._verify_before_entry("nano", [10 * NANOSECOND])
