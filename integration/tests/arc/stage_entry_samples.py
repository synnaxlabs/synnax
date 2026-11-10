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
SEQUENCES = ["drop", "straddle", "counter"]

SECOND = sy.TimeSpan.SECOND
MILLISECOND = sy.TimeSpan.MILLISECOND
# The value every sample taken before entry carries. Seeing it on pt_seen is a leak.
BEFORE_ENTRY = -1.0
# The time between each pt's first sample and stage entry. Samples taken before entry
# are stamped in this gap, after the first sample, so their writes do not overlap it.
# The runtime forgets a clock offset after 5 to 10 s, so the gap stays well below that.
HISTORY_GAP = 1 * SECOND
# How long after the first sample the samples taken before entry start.
STALE_AFTER_HISTORY = 100 * MILLISECOND
# The counter sequence mirrors a field abort: 50 samples of persistence at 5 kHz on a
# 300 psi low limit, armed on entering stage B.
COUNTER_SAMPLES = 100
COUNTER_PERIOD = 200 * sy.TimeSpan.MICROSECOND
COUNTER_IN_STAGE_FROM = 50 * MILLISECOND
COUNTER_LAST = 333.0


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


def _counter_func(tag: str) -> str:
    """A per-sample limit check with a stateful sample counter, as a field sequence
    writes one: config params, a flow-fed value, and channel writes from the body."""
    idx, reason = _ch("counter", "abort_idx", tag), _ch("counter", "abort_reason", tag)
    return (
        "func limit_persists(hi f64, lo f64, idx i32, n i64) {value f64} u8 {\n"
        "    count $= 0\n"
        "    alarm_state u8 $= 0\n"
        "    if value > hi or value < lo {\n"
        "        count = count + 1\n"
        "    } else {\n"
        "        count = 0\n"
        "    }\n"
        "    if count >= n {\n"
        "        count = 0\n"
        "        alarm_state = 1\n"
        f"        {idx} = idx\n"
        f'        {reason} = "ABORT: LOW" + f" {{value:.2f}} PSIA"\n'
        "    }\n"
        "    return alarm_state\n"
        "}\n"
    )


def _counter_sequence(tag: str) -> str:
    p = "counter"
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
        {pt} -> limit_persists(hi=1000, lo=300, idx=7045, n=50) => aborted
    }}
    stage aborted {{
        "aborted" -> {log}
    }}
}}
{START_CMD} => {p}_main
"""


class StageEntrySamples(ArcCase):
    """A stage sees only samples taken after it was entered (SY-5456), once the
    channel's index has learned its clock offset from earlier data (SY-5771).

    The test stamps every `pt` sample itself; samples taken before entry carry
    BEFORE_ENTRY. Each `pt` first gets one sample on arrival, before its stage is
    entered. `entered` carries the entry cycle stamp. A `probe` seen after a
    pre-entry sample proves the runtime already handled it, so a leak would show.
    """

    arc_name_prefix = "StageEntrySamples"
    start_cmd_channel = START_CMD

    def setup(self) -> None:
        self.tag = random_name()
        self._history: dict[str, sy.TimeStamp] = {}
        self.arc_source = _counter_func(self.tag) + "".join(
            _counter_sequence(self.tag) if p == "counter" else _sequence(p, self.tag)
            for p in SEQUENCES
        )
        self.subscribe_channels = (
            [
                self._ch(p, n)
                for p in SEQUENCES
                for n in ("log", "pt_seen", "probe_seen")
            ]
            + [self._entered_time(p) for p in SEQUENCES]
            + [self._ch("counter", "abort_idx"), self._ch("counter", "abort_reason")]
        )
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
        create_virtual_channels(
            self.client,
            [
                (self._ch("counter", "abort_idx"), sy.DataType.INT32),
                (self._ch("counter", "abort_reason"), sy.DataType.STRING),
            ],
        )
        super().setup()

    def _ch(self, p: str, name: str) -> str:
        return _ch(p, name, self.tag)

    def _entered_time(self, p: str) -> str:
        return f"{self._ch(p, 'entered')}_time"

    def _enter_b(self, p: str) -> sy.TimeStamp:
        """Give pt its first sample, move the sequence into stage B, and return its
        entry cycle stamp."""
        self.wait_for_eq(self._ch(p, "log"), "a")
        self._history[p] = self._write_history(p)
        sy.sleep(HISTORY_GAP)  # noqa: TID251 leaves room for stale samples
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

    def _write_history(self, p: str) -> sy.TimeStamp:
        """Write one pt sample stamped now, so its index learns its clock offset."""
        now = sy.TimeStamp.now()
        self._write_pt(p, [now], [BEFORE_ENTRY])
        return now

    def _stale(self, p: str, n: int, period: sy.TimeSpan) -> list[sy.TimeStamp]:
        """Return n stamps taken before entry, after the first pt sample."""
        first = self._history[p] + STALE_AFTER_HISTORY
        return [first + period * i for i in range(n)]

    def _verify_before_entry(self) -> None:
        """Deliver samples taken before entry; none may reach B."""
        p = "drop"
        self.log(f"[{p}] Samples taken before entry must never reach B")
        self._enter_b(p)
        self._write_pt(p, self._stale(p, 3, 100 * MILLISECOND), [BEFORE_ENTRY] * 3)
        self._expect_not_leaked(p)

    def _verify_straddle(self) -> None:
        self.log("[straddle] Only the samples taken at or after entry reach B")
        entered = self._enter_b("straddle")
        self._write_pt(
            "straddle",
            [
                *self._stale("straddle", 2, 100 * MILLISECOND),
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

    def _verify_counter(self) -> None:
        """The batch in flight at entry, 100 samples below the limit, must not trip a
        50-sample counter armed on entry. The samples taken inside B are above it."""
        p = "counter"
        self.log(f"[{p}] A stale batch must not trip a counter armed on entry")
        entered = self._enter_b(p)
        stale = self._stale(p, COUNTER_SAMPLES, COUNTER_PERIOD)
        stale_values = [255.0 + 0.35 * i for i in range(COUNTER_SAMPLES - 1)] + [289.6]
        self._write_pt(p, stale, stale_values)
        in_stage = entered + COUNTER_IN_STAGE_FROM
        inside = [in_stage + COUNTER_PERIOD * i for i in range(COUNTER_SAMPLES)]
        in_values = [313.0 + 0.2 * i for i in range(COUNTER_SAMPLES - 1)] + [
            COUNTER_LAST
        ]
        self._write_pt(p, inside, in_values)
        # An abort on the stale batch leaves B, so the probe never comes back.
        self.writer.write(self._ch(p, "probe"), 1)
        try:
            self.wait_for_eq(self._ch(p, "probe_seen"), 1)
        except RuntimeError:
            self.fail(
                f"stage B of {p}_main was left: "
                f"log={self.read_tlm(self._ch(p, 'log'))!r} "
                f"last seen={self.read_tlm(self._ch(p, 'pt_seen'))!r} "
                f"abort_idx={self.read_tlm(self._ch(p, 'abort_idx'))!r} "
                f"abort_reason={self.read_tlm(self._ch(p, 'abort_reason'))!r}"
            )
        self.wait_for_eq(self._ch(p, "pt_seen"), COUNTER_LAST)

    def verify_sequence_execution(self) -> None:
        self._verify_counter()
        self._verify_before_entry()
        self._verify_straddle()
