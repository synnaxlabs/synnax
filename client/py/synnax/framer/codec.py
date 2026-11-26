#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import struct
from typing import override

import numpy as np

import synnax.channel.payload as channel
from synnax.exceptions import ValidationError
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import Alignment, DataType, Series, TimeRange
from x.codec import Codec as XCodec, JSONCodec
from x.telem import DataTypeCode, code_of, data_type_of

_EXTENDED_FLAG_POS = 7
_MULTIPLES_PRESENT_FLAG_POS = 6
_ZERO_ALIGNMENTS_FLAG_POS = 5
_EQUAL_ALIGNMENTS_FLAG_POS = 4
_EQUAL_LENGTHS_FLAG_POS = 3
_EQUAL_TIME_RANGES_FLAG_POS = 2
_TIME_RANGES_ZERO_FLAG_POS = 1
_ALL_CHANNELS_PRESENT_FLAG_POS = 0

_TIME_RANGE_SIZE = 16
_U8 = struct.Struct("<B")
_U32 = struct.Struct("<I")
_U64 = struct.Struct("<Q")
_TIME_RANGE = struct.Struct("<QQ")
_ZERO_ALIGNMENT = Alignment(0)
_ALIGNMENT_SIZE = 8
_MULTIPLE_SIZE = 4
_DATA_LENGTH_SIZE = 4
_KEY_SIZE = 4
_FLAGS_SIZE = 1
_SEQ_NUM_SIZE = 4
_BITS_PER_BYTE = 8


def _bit_packed_byte_count(n_samples: int) -> int:
    return (n_samples + _BITS_PER_BYTE - 1) // _BITS_PER_BYTE


def _time_range_nanoseconds(tr: TimeRange | None) -> tuple[int, int]:
    """Returns the start and end of tr in nanoseconds, with no time range as zero."""
    return (0, 0) if tr is None else (int(tr.start), int(tr.end))


def _series_wire_byte_length(ser: Series) -> int:
    if ser.data_type == DataType.BOOLEAN:
        return _bit_packed_byte_count(len(ser))
    return len(ser.data)


def _pack_bool_bits(src: bytes) -> bytes:
    nonzero = np.frombuffer(src, dtype=np.uint8) != 0
    return np.packbits(nonzero, bitorder="little").tobytes()


def _unpack_bool_bits(src: memoryview, sample_count: int) -> bytes:
    packed = np.frombuffer(src, dtype=np.uint8)
    return np.unpackbits(packed, count=sample_count, bitorder="little").tobytes()


_TYPES_DECLARED = 1 << 0
_OVERRIDES_PRESENT = 1 << 1
_CLOCKS_DECLARED = 1 << 3
# The extension bits and override kinds that no codec sets yet.
_RESERVED_EXTENSION_FLAGS = 0b1111_0100
_TYPE_KIND = 1 << 0
_CLOCK_KIND = 1 << 1
_RESERVED_KINDS = 0b1111_1100
# The next override ordinal that ends the chain.
_NO_OVERRIDE = 0xFFFFFFFF


def _is_int64_like(dt: DataType) -> bool:
    return dt == DataType.INT64 or dt == DataType.TIMESTAMP


def _needs_type_override(agreed: DataType, dt: DataType) -> bool:
    return agreed != dt and not (_is_int64_like(agreed) and _is_int64_like(dt))


def _code_of(key: channel.Key, dt: DataType) -> int:
    code = code_of(dt)
    if code == DataTypeCode.unknown:
        raise ValidationError(f'data type "{dt}" of channel {key} has no type code')
    return int(code)


class _Agreement:
    """The values that the frames of one state agree on, as one side sees them."""

    def __init__(self, keys: list[channel.Key], data_types: list[DataType]) -> None:
        self.types_declared = False
        # dropped is true once a frame of the state failed to decode.
        self.dropped = False
        self.data_types = data_types
        # The agreed clock of each requested key. Zero is no clock.
        self.clocks = [0] * len(keys)
        self.declared: list[channel.Key] = []
        # The ascending union of the requested and the declared keys.
        self.all = keys


class CodecState:
    keys: list[channel.Key]
    index: dict[channel.Key, int]

    def __init__(self, *, keys: list[channel.Key], data_types: list[DataType]) -> None:
        self.keys = sorted(set(keys))
        self.index = {k: i for i, k in enumerate(self.keys)}
        agreed = [DataType.UNKNOWN] * len(self.keys)
        for k, dt in zip(keys, data_types, strict=True):
            agreed[self.index[k]] = dt
        self.enc = _Agreement(self.keys, agreed)
        self.dec = _Agreement(self.keys, [DataType.UNKNOWN] * len(self.keys))


class _Plan:
    def __init__(self, ag: _Agreement) -> None:
        self.ext = 0
        self.codes: list[int] = []
        self.kinds: list[int] = []
        self.clocks = ag.clocks
        self.declared = ag.declared
        self.all = ag.all


class _Reader:
    """Reads a frame, failing on any read past its end."""

    __slots__ = ("buf", "end", "idx")

    def __init__(self, data: memoryview, offset: int) -> None:
        self.buf = data
        self.end = len(data)
        self.idx = offset

    @property
    def remaining(self) -> int:
        return self.end - self.idx

    def _advance(self, n: int) -> int:
        """Returns the offset of the next n bytes and moves past them."""
        i = self.idx
        if i + n > self.end:
            raise ValidationError("remote sent a frame that ends early")
        self.idx = i + n
        return i

    def unpack(self, fmt: struct.Struct) -> tuple[int, ...]:
        return fmt.unpack_from(self.buf, self._advance(fmt.size))

    def uint8(self) -> int:
        return self.buf[self._advance(1)]

    def uint32(self) -> int:
        v: int = _U32.unpack_from(self.buf, self._advance(4))[0]
        return v

    def uint64(self) -> int:
        v: int = _U64.unpack_from(self.buf, self._advance(8))[0]
        return v

    def bytes(self, n: int) -> memoryview:
        i = self._advance(n)
        return self.buf[i : i + n]

    def data_type(self) -> DataType:
        code = self.uint8()
        dt = data_type_of(code)
        if dt == DataType.UNKNOWN:
            raise ValidationError(
                f"remote sent data type code {code}, which this codec does not know"
            )
        return dt


def _union(a: list[channel.Key], b: list[channel.Key]) -> list[channel.Key]:
    return a if len(b) == 0 else sorted(a + b)


class Codec:
    _seq_num: int
    _states: dict[int, CodecState]
    _curr_state: CodecState | None = None

    def __init__(
        self,
        *,
        keys: list[channel.Key] | None = None,
        data_types: list[DataType] | None = None,
    ) -> None:
        self._seq_num = 0
        self._states = {}
        if keys is not None and data_types is not None:
            self.update(keys=keys, data_types=data_types)

    def update(self, *, keys: list[channel.Key], data_types: list[DataType]) -> None:
        """Starts a new state of the given keys and their data types. The data types
        are the encoder's agreed types; the decoder reads its own from the first frame
        of the state."""
        self._seq_num += 1
        self._curr_state = CodecState(keys=keys, data_types=data_types)
        self._states[self._seq_num] = self._curr_state

    def throw_if_not_updated(self, op_name: str) -> None:
        if self._curr_state is None:
            raise ValueError(
                "Codec has not been updated with keys and data types. "
                f"Please call update() before calling {op_name}()."
            )

    @staticmethod
    def _entries(
        pld: FramePayload, state: CodecState
    ) -> list[tuple[channel.Key, Series, int]]:
        entries = []
        for key, ser in zip(pld.keys, pld.series, strict=True):
            i = state.index.get(key)
            if i is not None:
                entries.append((key, ser, i))
            elif ser.clock == key and _is_int64_like(ser.data_type):
                entries.append((key, ser, -1))
            else:
                raise ValidationError(
                    f"encoder was provided a key {key} that is not in the codec state."
                )
        # A stable sort keeps the producer's order within a key.
        entries.sort(key=lambda e: e[0])
        return entries

    @staticmethod
    def _plan(
        entries: list[tuple[channel.Key, Series, int]], state: CodecState
    ) -> _Plan:
        ag = state.enc
        plan = _Plan(ag)
        if not ag.types_declared:
            plan.ext |= _TYPES_DECLARED
            plan.codes = [
                _code_of(k, dt) for k, dt in zip(state.keys, ag.data_types, strict=True)
            ]
        overrides = undeclared = False
        for _, ser, i in entries:
            if i < 0:
                undeclared = True
            elif ser.clock != ag.clocks[i] or _needs_type_override(
                ag.data_types[i], ser.data_type
            ):
                overrides = True
        if overrides:
            plan.ext |= _OVERRIDES_PRESENT
            plan.clocks = list(ag.clocks)
            for key, ser, i in entries:
                kinds = 0
                if i >= 0:
                    if _needs_type_override(ag.data_types[i], ser.data_type):
                        _code_of(key, ser.data_type)
                        kinds |= _TYPE_KIND
                    if ser.clock != plan.clocks[i]:
                        kinds |= _CLOCK_KIND
                        plan.clocks[i] = ser.clock
                plan.kinds.append(kinds)
        if undeclared or len(ag.declared) > 0:
            in_frame = {k for k, _, i in entries if i < 0}
            declared = sorted(
                in_frame
                | {k for k in ag.declared if k not in in_frame and k in plan.clocks}
            )
            if declared != ag.declared:
                plan.ext |= _CLOCKS_DECLARED
                plan.declared = declared
                plan.all = _union(state.keys, declared)
        return plan

    def encode(self, frame: Frame | FramePayload, *, start_offset: int = 0) -> bytes:
        self.throw_if_not_updated("encode")
        assert self._curr_state is not None
        state = self._curr_state
        pld = frame if isinstance(frame, FramePayload) else frame.to_payload()
        entries = self._entries(pld, state)
        plan = self._plan(entries, state)
        n = len(entries)

        eq_len = True
        eq_tr = True
        eq_align = True
        multiples_present = False
        ref_len = -1
        ref_tr = (0, 0)
        ref_align = 0
        size = start_offset + _FLAGS_SIZE + _SEQ_NUM_SIZE
        all_present = [k for k, _, _ in entries] == plan.all
        if not all_present:
            size += n * _KEY_SIZE
        for _, ser, _ in entries:
            tr = _time_range_nanoseconds(ser.time_range)
            if ser.data_type.is_variable:
                eq_len = False
            if ser.alignment_multiple > 1:
                multiples_present = True
            size += _series_wire_byte_length(ser)
            if ref_len == -1:
                ref_len, ref_tr, ref_align = len(ser), tr, int(ser.alignment)
                continue
            if len(ser) != ref_len:
                eq_len = False
            if tr != ref_tr:
                eq_tr = False
            if int(ser.alignment) != ref_align:
                eq_align = False
        tr_zero = eq_tr and ref_tr == (0, 0)
        zero_align = eq_align and ref_align == 0
        size += (1 if eq_len else n) * _DATA_LENGTH_SIZE
        if not tr_zero:
            size += (1 if eq_tr else n) * _TIME_RANGE_SIZE
        if not zero_align:
            size += (1 if eq_align else n) * _ALIGNMENT_SIZE
        if multiples_present:
            size += n * _MULTIPLE_SIZE
        if plan.ext != 0:
            size += 1 + len(plan.codes)
        if plan.ext & _CLOCKS_DECLARED:
            size += _KEY_SIZE * (1 + len(plan.declared))
        if plan.ext & _OVERRIDES_PRESENT:
            size += 4
        for kinds in plan.kinds:
            if kinds:
                size += 5 + (1 if kinds & _TYPE_KIND else 0)
                size += 4 if kinds & _CLOCK_KIND else 0

        buf = bytearray(size)
        off = start_offset

        def put(fmt: struct.Struct, *v: int) -> None:
            nonlocal off
            fmt.pack_into(buf, off, *v)
            off += fmt.size

        flags = (
            (int(plan.ext != 0) << _EXTENDED_FLAG_POS)
            | (int(multiples_present) << _MULTIPLES_PRESENT_FLAG_POS)
            | (int(zero_align) << _ZERO_ALIGNMENTS_FLAG_POS)
            | (int(eq_align) << _EQUAL_ALIGNMENTS_FLAG_POS)
            | (int(eq_len) << _EQUAL_LENGTHS_FLAG_POS)
            | (int(eq_tr) << _EQUAL_TIME_RANGES_FLAG_POS)
            | (int(tr_zero) << _TIME_RANGES_ZERO_FLAG_POS)
            | (int(all_present) << _ALL_CHANNELS_PRESENT_FLAG_POS)
        )
        put(_U8, flags)
        put(_U32, self._seq_num)
        if plan.ext != 0:
            put(_U8, plan.ext)
            for code in plan.codes:
                put(_U8, code)
        if plan.ext & _CLOCKS_DECLARED:
            put(_U32, len(plan.declared))
            for k in plan.declared:
                put(_U32, k)
        if eq_len:
            put(_U32, max(ref_len, 0) if ref_len != -1 else 0xFFFFFFFF)
        if eq_tr and not tr_zero:
            put(_TIME_RANGE, *ref_tr)
        if eq_align and not zero_align:
            put(_U64, ref_align)

        def next_override(start: int) -> int:
            for i in range(start, len(plan.kinds)):
                if plan.kinds[i]:
                    return i
            return _NO_OVERRIDE

        if plan.ext & _OVERRIDES_PRESENT:
            put(_U32, next_override(0))
        for i, (key, ser, _) in enumerate(entries):
            if not all_present:
                put(_U32, key)
            kinds = plan.kinds[i] if i < len(plan.kinds) else 0
            if kinds:
                put(_U8, kinds)
                if kinds & _TYPE_KIND:
                    put(_U8, int(code_of(ser.data_type)))
                if kinds & _CLOCK_KIND:
                    put(_U32, ser.clock)
                put(_U32, next_override(i + 1))
            if not eq_len:
                put(_U32, len(ser.data) if ser.data_type.is_variable else len(ser))
            data = (
                _pack_bool_bits(ser.data)
                if ser.data_type == DataType.BOOLEAN
                else ser.data
            )
            buf[off : off + len(data)] = data
            off += len(data)
            if not eq_tr:
                put(_TIME_RANGE, *_time_range_nanoseconds(ser.time_range))
            if not eq_align:
                put(_U64, int(ser.alignment))
            if multiples_present:
                put(_U32, ser.alignment_multiple)
        ag = state.enc
        ag.types_declared = True
        ag.clocks, ag.declared, ag.all = plan.clocks, plan.declared, plan.all
        return bytes(buf)

    def decode(self, data: bytes, *, offset: int = 0) -> FramePayload:
        """Decodes a frame. A frame that fails to decode drops its state, so every
        later frame of the state fails."""
        self.throw_if_not_updated("decode")
        r = _Reader(memoryview(data), offset)
        flags = r.uint8()
        seq_num = r.uint32()
        state = self._states.get(seq_num)
        if state is None:
            raise ValidationError(f"remote sent invalid sequence number {seq_num}")
        # The remote never encodes with an older state again.
        if len(self._states) > 1:
            for old in [s for s in self._states if s < seq_num]:
                del self._states[old]
        if state.dec.dropped:
            raise ValidationError(
                f"remote sent a frame for state {seq_num} after a frame of it failed "
                "to decode"
            )
        try:
            return self._decode(r, flags, seq_num, state)
        except BaseException:
            state.dec.dropped = True
            raise

    @staticmethod
    def _decode(
        r: _Reader, flags: int, seq_num: int, state: CodecState
    ) -> FramePayload:
        eq_len = (flags >> _EQUAL_LENGTHS_FLAG_POS) & 1 == 1
        eq_tr = (flags >> _EQUAL_TIME_RANGES_FLAG_POS) & 1 == 1
        tr_zero = (flags >> _TIME_RANGES_ZERO_FLAG_POS) & 1 == 1
        eq_align = (flags >> _EQUAL_ALIGNMENTS_FLAG_POS) & 1 == 1
        zero_align = (flags >> _ZERO_ALIGNMENTS_FLAG_POS) & 1 == 1
        multiples_present = (flags >> _MULTIPLES_PRESENT_FLAG_POS) & 1 == 1
        all_present = (flags >> _ALL_CHANNELS_PRESENT_FLAG_POS) & 1 == 1
        ag = state.dec
        ext = 0
        if (flags >> _EXTENDED_FLAG_POS) & 1 == 1:
            ext = r.uint8()
            if ext == 0:
                raise ValidationError("remote sent extension flags of 0")
            if ext & _RESERVED_EXTENSION_FLAGS:
                raise ValidationError(
                    f"remote sent extension flags {ext:08b}, which hold a reserved bit"
                )
        if not ext & _TYPES_DECLARED:
            if not ag.types_declared:
                raise ValidationError(
                    f"remote sent no type declaration for state {seq_num}, so it runs "
                    "an older codec, which this codec does not support"
                )
        else:
            if ag.types_declared:
                raise ValidationError(
                    f"remote declared the types of state {seq_num} twice"
                )
            ag.data_types = [r.data_type() for _ in state.keys]
            ag.types_declared = True
        if ext & _CLOCKS_DECLARED:
            n = r.uint32()
            if n > r.remaining // _KEY_SIZE:
                raise ValidationError("remote sent a frame that ends early")
            declared: list[channel.Key] = []
            for i in range(n):
                key = r.uint32()
                if key in state.index:
                    raise ValidationError(
                        f"remote declared clock {key}, which the request names"
                    )
                if i > 0 and key <= declared[-1]:
                    raise ValidationError(
                        f"remote declared clock {key} after clock {declared[-1]}, "
                        "so the list is not ascending"
                    )
                declared.append(key)
            ag.declared = declared
            ag.all = _union(state.keys, declared)
        shared_len = r.uint32() if eq_len else 0
        tr = TimeRange.ZERO
        if eq_tr and not tr_zero:
            start, end = r.unpack(_TIME_RANGE)
            tr = TimeRange(start=start, end=end)
        alignment = (
            Alignment(r.uint64()) if eq_align and not zero_align else _ZERO_ALIGNMENT
        )
        nxt = r.uint32() if ext & _OVERRIDES_PRESENT else _NO_OVERRIDE
        ordinal = 0

        def read_kinds() -> int:
            kinds = r.uint8()
            if kinds == 0:
                raise ValidationError(
                    f"remote sent override kinds of 0 for series {ordinal}"
                )
            if kinds & _RESERVED_KINDS:
                raise ValidationError(
                    f"remote sent override kinds {kinds:08b} for series {ordinal}, "
                    "which hold a reserved bit"
                )
            return kinds

        def agreement_of(key: channel.Key) -> tuple[DataType, int]:
            nonlocal nxt
            i = state.index.get(key)
            if i is None:
                if key not in ag.declared:
                    raise ValidationError(f"unknown channel key: {key}")
                if ordinal == nxt:
                    read_kinds()
                    raise ValidationError(
                        f"remote sent an override for declared clock {key}"
                    )
                return DataType.TIMESTAMP, key
            dt = ag.data_types[i]
            if ordinal == nxt:
                kinds = read_kinds()
                if kinds & _TYPE_KIND:
                    dt = r.data_type()
                if kinds & _CLOCK_KIND:
                    ag.clocks[i] = r.uint32()
                nxt = r.uint32()
                if nxt != _NO_OVERRIDE and nxt <= ordinal:
                    raise ValidationError(
                        f"remote sent next override {nxt}, which is not after series "
                        f"{ordinal}"
                    )
            return dt, ag.clocks[i]

        keys: list[channel.Key] = []
        series: list[Series] = []

        def decode_series(key: channel.Key) -> None:
            nonlocal ordinal
            dt, clock = agreement_of(key)
            ordinal += 1
            if eq_len and dt.is_variable:
                raise ValidationError(
                    f"remote sent variable data type {dt} under a shared length"
                )
            length = shared_len if eq_len else r.uint32()
            if dt == DataType.BOOLEAN:
                data = _unpack_bool_bits(
                    r.bytes(_bit_packed_byte_count(length)), length
                )
            else:
                n = length if dt.is_variable else length * int(dt.density)
                data = bytes(r.bytes(n))
            s_tr = tr
            if not eq_tr:
                start, end = r.unpack(_TIME_RANGE)
                s_tr = TimeRange(start=start, end=end)
            s_align = Alignment(r.uint64()) if not eq_align else alignment
            multiple = max(r.uint32(), 1) if multiples_present else 1
            keys.append(key)
            series.append(
                Series(
                    data_type=dt,
                    data=data,
                    time_range=s_tr,
                    alignment=s_align,
                    alignment_multiple=multiple,
                    clock=clock,
                )
            )

        if all_present:
            for key in ag.all:
                decode_series(key)
            if r.remaining > 0:
                raise ValidationError("remote sent bytes after the last series")
        else:
            while r.remaining > 0:
                decode_series(r.uint32())
        if nxt != _NO_OVERRIDE:
            raise ValidationError(
                f"remote ended the frame with the override for series {nxt} pending"
            )
        return FramePayload(keys=keys, series=series)


LOW_PERF_SPECIAL_CHAR = 254
HIGH_PERF_SPECIAL_CHAR = 255


class WSCodec(XCodec):
    def __init__(self, codec: Codec) -> None:
        self.codec = codec
        self.lower_perf_codec = JSONCodec()

    @override
    def content_type(self) -> str:
        return "application/vnd.synnax.frame"
