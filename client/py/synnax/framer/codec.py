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

import synnax.channel.payload as channel
from synnax.exceptions import ValidationError
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import Alignment, DataType, Series, TimeRange
from x.codec import Codec as XCodec
from x.codec import JSONCodec

_EXTENDED_FLAG_POS = 7
_MULTIPLES_PRESENT_FLAG_POS = 6
_ZERO_ALIGNMENTS_FLAG_POS = 5
_EQUAL_ALIGNMENTS_FLAG_POS = 4
_EQUAL_LENGTHS_FLAG_POS = 3
_EQUAL_TIME_RANGES_FLAG_POS = 2
_TIME_RANGES_ZERO_FLAG_POS = 1
_ALL_CHANNELS_PRESENT_FLAG_POS = 0

_TIME_RANGE_SIZE = 16
_ALIGNMENT_SIZE = 8
_MULTIPLE_SIZE = 4
_DATA_LENGTH_SIZE = 4
_KEY_SIZE = 4
_FLAGS_SIZE = 1
_SEQ_NUM_SIZE = 4
_BITS_PER_BYTE = 8


def _bit_packed_byte_count(n_samples: int) -> int:
    return (n_samples + _BITS_PER_BYTE - 1) // _BITS_PER_BYTE


def _series_wire_byte_length(ser: Series) -> int:
    if ser.data_type == DataType.BOOLEAN:
        return _bit_packed_byte_count(len(ser))
    return len(ser.data)


def _pack_bool_bits(src: bytes) -> bytes:
    dst = bytearray(_bit_packed_byte_count(len(src)))
    for i, b in enumerate(src):
        if b != 0:
            dst[i // _BITS_PER_BYTE] |= 1 << (i % _BITS_PER_BYTE)
    return bytes(dst)


def _unpack_bool_bits(src: memoryview, sample_count: int) -> bytes:
    dst = bytearray(sample_count)
    for i in range(sample_count):
        dst[i] = (src[i // _BITS_PER_BYTE] >> (i % _BITS_PER_BYTE)) & 1
    return bytes(dst)


class _CodecFlags:
    def __init__(self) -> None:
        self.eq_len: bool = True
        self.eq_tr: bool = True
        self.time_ranges_zero: bool = True
        self.all_channels_present: bool = True
        self.eq_align: bool = True
        self.zero_alignments: bool = True
        self.multiples_present: bool = False

    def encode(self) -> int:
        b = 0
        if self.eq_len:
            b |= 1 << _EQUAL_LENGTHS_FLAG_POS
        if self.eq_tr:
            b |= 1 << _EQUAL_TIME_RANGES_FLAG_POS
        if self.time_ranges_zero:
            b |= 1 << _TIME_RANGES_ZERO_FLAG_POS
        if self.all_channels_present:
            b |= 1 << _ALL_CHANNELS_PRESENT_FLAG_POS
        if self.eq_align:
            b |= 1 << _EQUAL_ALIGNMENTS_FLAG_POS
        if self.zero_alignments:
            b |= 1 << _ZERO_ALIGNMENTS_FLAG_POS
        if self.multiples_present:
            b |= 1 << _MULTIPLES_PRESENT_FLAG_POS
        return b

    @classmethod
    def decode(cls, b: int) -> _CodecFlags:
        if (b >> _EXTENDED_FLAG_POS) & 1:
            raise ValidationError(
                "remote sent an extended flags byte, which this codec does not support"
            )
        flags = cls()
        flags.eq_len = bool((b >> _EQUAL_LENGTHS_FLAG_POS) & 1)
        flags.eq_tr = bool((b >> _EQUAL_TIME_RANGES_FLAG_POS) & 1)
        flags.time_ranges_zero = bool((b >> _TIME_RANGES_ZERO_FLAG_POS) & 1)
        flags.all_channels_present = bool((b >> _ALL_CHANNELS_PRESENT_FLAG_POS) & 1)
        flags.eq_align = bool((b >> _EQUAL_ALIGNMENTS_FLAG_POS) & 1)
        flags.zero_alignments = bool((b >> _ZERO_ALIGNMENTS_FLAG_POS) & 1)
        flags.multiples_present = bool((b >> _MULTIPLES_PRESENT_FLAG_POS) & 1)
        return flags


class CodecState:
    keys: list[channel.Key]
    data_types: dict[channel.Key, DataType]
    has_variable_data_types: bool

    def __init__(self, *, keys: list[channel.Key], data_types: list[DataType]) -> None:
        self.keys = sorted(keys)
        self.data_types = dict(zip(keys, data_types, strict=True))
        self.has_variable_data_types = any(dt.is_variable for dt in data_types)


class Codec:
    _has_variable_data_types: bool
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
        self._seq_num += 1
        self._curr_state = CodecState(keys=keys, data_types=data_types)
        self._states[self._seq_num] = self._curr_state

    def throw_if_not_updated(self, op_name: str) -> None:
        if self._curr_state is None:
            raise ValueError(
                "Codec has not been updated with keys and data types. "
                f"Please call update() before calling {op_name}()."
            )

    def encode(self, frame: Frame | FramePayload, *, start_offset: int = 0) -> bytes:
        self.throw_if_not_updated("encode")
        assert self._curr_state is not None
        pld = frame if isinstance(frame, FramePayload) else frame.to_payload()
        indices = sorted(range(len(pld.keys)), key=lambda i: pld.keys[i])
        sorted_keys = [pld.keys[i] for i in indices]
        sorted_series = [pld.series[i] for i in indices]

        flg = _CodecFlags()
        flg.eq_len = not self._curr_state.has_variable_data_types
        curr_data_size = -1
        ref_tr = None
        ref_align = None
        byte_array_size = start_offset + _FLAGS_SIZE + _SEQ_NUM_SIZE

        if len(sorted_keys) != len(self._curr_state.keys):
            flg.all_channels_present = False
            byte_array_size += len(sorted_keys) * _KEY_SIZE

        for i, ser in enumerate(sorted_series):
            key = sorted_keys[i]
            dt = self._curr_state.data_types.get(key, None)
            if dt is None:
                raise ValidationError(
                    f"encoder was provided a key {key} that is not in the codec state."
                )
            if dt != ser.data_type:
                raise ValidationError(
                    f"data type {ser.data_type} for key {key} does not match codec "
                    f"state data type {dt}."
                )
            if curr_data_size == -1:
                curr_data_size = len(ser)
                ref_tr = ser.time_range
                ref_align = ser.alignment
            else:
                if len(ser) != curr_data_size:
                    flg.eq_len = False
                if ser.time_range != ref_tr:
                    flg.eq_tr = False
                if ser.alignment != ref_align:
                    flg.eq_align = False
            if ser.alignment_multiple > 1:
                flg.multiples_present = True

            byte_array_size += _series_wire_byte_length(ser)

        flg.time_ranges_zero = ref_tr is None or ref_tr == 0
        flg.zero_alignments = flg.eq_align and ref_align == 0
        byte_array_size += (int(flg.eq_len) or len(sorted_keys)) * _DATA_LENGTH_SIZE
        if not flg.time_ranges_zero:
            byte_array_size += (int(flg.eq_tr) or len(sorted_keys)) * _TIME_RANGE_SIZE
        if not flg.zero_alignments:
            byte_array_size += (int(flg.eq_align) or len(sorted_keys)) * _ALIGNMENT_SIZE
        if flg.multiples_present:
            byte_array_size += len(sorted_keys) * _MULTIPLE_SIZE

        buffer = bytearray(byte_array_size)
        offset = start_offset

        buffer[offset] = flg.encode()
        offset += _FLAGS_SIZE

        struct.pack_into("<I", buffer, offset, self._seq_num)
        offset += _SEQ_NUM_SIZE

        if flg.eq_len:
            struct.pack_into("<I", buffer, offset, curr_data_size)
            offset += _DATA_LENGTH_SIZE

        if flg.eq_tr and not flg.time_ranges_zero and ref_tr:
            struct.pack_into("<QQ", buffer, offset, ref_tr.start, ref_tr.end)
            offset += _TIME_RANGE_SIZE

        if flg.eq_align and not flg.zero_alignments and ref_align:
            struct.pack_into("<Q", buffer, offset, ref_align)
            offset += _ALIGNMENT_SIZE

        for i, ser in enumerate(sorted_series):
            k = sorted_keys[i]
            dt = self._curr_state.data_types.get(k, None)
            if dt is None:
                raise ValidationError(
                    f"Codec state does not contain data type for key {k}."
                )
            if dt != ser.data_type:
                raise ValidationError(
                    f"Series data type {ser.data_type} for key {k} does not match "
                    f"codec state data type {dt}."
                )

            if not flg.all_channels_present:
                struct.pack_into("<I", buffer, offset, sorted_keys[i])
                offset += _KEY_SIZE

            if not flg.eq_len:
                len_or_size = len(ser)
                if ser.data_type.is_variable:
                    len_or_size = len(ser.data)
                struct.pack_into("<I", buffer, offset, len_or_size)
                offset += _DATA_LENGTH_SIZE

            if ser.data_type == DataType.BOOLEAN:
                packed = _pack_bool_bits(ser.data)
                buffer[offset : offset + len(packed)] = packed
                offset += len(packed)
            else:
                buffer[offset : offset + len(ser.data)] = ser.data
                offset += len(ser.data)

            if not flg.eq_tr and not flg.time_ranges_zero:
                tr = TimeRange.ZERO if ser.time_range is None else ser.time_range
                struct.pack_into("<QQ", buffer, offset, tr.start, tr.end)
                offset += _TIME_RANGE_SIZE

            if not flg.eq_align and not flg.zero_alignments:
                struct.pack_into("<Q", buffer, offset, ser.alignment)
                offset += _ALIGNMENT_SIZE

            if flg.multiples_present:
                struct.pack_into("<I", buffer, offset, ser.alignment_multiple)
                offset += _MULTIPLE_SIZE

        return bytes(buffer)

    def decode(self, data: bytes, *, offset: int = 0) -> FramePayload:
        self.throw_if_not_updated("decode")
        assert self._curr_state is not None
        buffer = memoryview(data)
        idx = offset
        flags = _CodecFlags.decode(buffer[idx])
        idx += 1

        curr_seq_num = struct.unpack_from("<I", buffer, idx)[0]
        idx += _SEQ_NUM_SIZE

        state = self._states.get(curr_seq_num)
        if state is None:
            raise ValidationError(f"remote sent invalid sequence number {curr_seq_num}")

        to_del = None
        for seq_num in self._states:
            if seq_num < curr_seq_num:
                if to_del is None:
                    to_del = set()
                to_del.add(seq_num)

        if to_del is not None:
            for seq_num in to_del:
                del self._states[seq_num]

        data_len = 0
        start_time = 0
        end_time = 0
        alignment = Alignment(0)

        if flags.eq_len:
            data_len = struct.unpack_from("<I", buffer, idx)[0]
            idx += _DATA_LENGTH_SIZE

        if flags.eq_tr and not flags.time_ranges_zero:
            start_time, end_time = struct.unpack_from("<QQ", buffer, idx)
            idx += _TIME_RANGE_SIZE

        if flags.eq_align and not flags.zero_alignments:
            alignment = Alignment(struct.unpack_from("<Q", buffer, idx)[0])
            idx += _ALIGNMENT_SIZE

        keys = []
        series_list = []

        def decode_series(key: channel.Key) -> bool:
            nonlocal idx
            data_type = state.data_types.get(key)
            if data_type is None:
                return False
            curr_len = data_len
            if not flags.eq_len:
                if idx + _DATA_LENGTH_SIZE > len(buffer):
                    return False
                curr_len = struct.unpack_from("<I", buffer, idx)[0]
                idx += _DATA_LENGTH_SIZE

            data_byte_len = curr_len
            if not data_type.is_variable:
                data_byte_len = curr_len * data_type.density

            if data_type == DataType.BOOLEAN:
                wire_bytes = _bit_packed_byte_count(curr_len)
                if idx + wire_bytes > len(buffer):
                    return False
                series_data = _unpack_bool_bits(
                    buffer[idx : idx + wire_bytes], curr_len
                )
                idx += wire_bytes
            else:
                if idx + data_byte_len > len(buffer):
                    return False
                series_data = bytes(buffer[idx : idx + data_byte_len])
                idx += data_byte_len

            if flags.time_ranges_zero:
                tr = TimeRange.ZERO
            elif flags.eq_tr:
                tr = TimeRange(start=start_time, end=end_time)
            else:
                if idx + _TIME_RANGE_SIZE > len(buffer):
                    return False
                s, e = struct.unpack_from("<QQ", buffer, idx)
                tr = TimeRange(start=s, end=e)
                idx += _TIME_RANGE_SIZE

            curr_alignment = alignment
            if not flags.eq_align and not flags.zero_alignments:
                if idx + _ALIGNMENT_SIZE > len(buffer):
                    return False
                curr_alignment = Alignment(struct.unpack_from("<Q", buffer, idx)[0])
                idx += _ALIGNMENT_SIZE

            multiple = 1
            if flags.multiples_present:
                if idx + _MULTIPLE_SIZE > len(buffer):
                    return False
                multiple = max(struct.unpack_from("<I", buffer, idx)[0], 1)
                idx += _MULTIPLE_SIZE

            keys.append(key)
            series_list.append(
                Series(
                    data_type=data_type,
                    data=series_data,
                    time_range=tr,
                    alignment=curr_alignment,
                    alignment_multiple=multiple,
                )
            )
            return True

        if flags.all_channels_present:
            for key in state.keys:
                if not decode_series(key):
                    break
        else:
            while idx + _KEY_SIZE <= len(buffer):
                frame_key = struct.unpack_from("<I", buffer, idx)[0]
                idx += _KEY_SIZE
                if not decode_series(frame_key):
                    break

        return FramePayload(keys=keys, series=series_list)


LOW_PERF_SPECIAL_CHAR = 254
HIGH_PERF_SPECIAL_CHAR = 255


class WSCodec(XCodec):
    def __init__(self, codec: Codec) -> None:
        self.codec = codec
        self.lower_perf_codec = JSONCodec()

    def content_type(self) -> str:
        return "application/vnd.synnax.frame"
