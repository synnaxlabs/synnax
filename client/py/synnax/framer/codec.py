#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import struct
from typing import Dict, List, Union

from freighter import JSONCodec
from freighter.codec import Codec as FreighterCodec

from synnax.channel.payload import ChannelKey, ChannelKeys
from synnax.exceptions import ValidationError
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import DataType, Series, TimeRange

ZERO_ALIGNMENTS_FLAG_POS = 5
EQUAL_ALIGNMENTS_FLAG_POS = 4
EQUAL_LENGTHS_FLAG_POS = 3
EQUAL_TIME_RANGES_FLAG_POS = 2
TIME_RANGES_ZERO_FLAG_POS = 1
ALL_CHANNELS_PRESENT_FLAG_POS = 0

TIME_RANGE_SIZE = 16
ALIGNMENT_SIZE = 8
DATA_LENGTH_SIZE = 4
KEY_SIZE = 4
FLAGS_SIZE = 1
SEQ_NUM_SIZE = 4


class CodecFlags:
    def __init__(self) -> None:
        self.eq_len: bool = True
        self.eq_tr: bool = True
        self.time_ranges_zero: bool = True
        self.all_channels_present: bool = True
        self.eq_align: bool = True
        self.zero_alignments: bool = True

    def encode(self) -> int:
        b = 0
        if self.eq_len:
            b |= 1 << EQUAL_LENGTHS_FLAG_POS
        if self.eq_tr:
            b |= 1 << EQUAL_TIME_RANGES_FLAG_POS
        if self.time_ranges_zero:
            b |= 1 << TIME_RANGES_ZERO_FLAG_POS
        if self.all_channels_present:
            b |= 1 << ALL_CHANNELS_PRESENT_FLAG_POS
        if self.eq_align:
            b |= 1 << EQUAL_ALIGNMENTS_FLAG_POS
        if self.zero_alignments:
            b |= 1 << ZERO_ALIGNMENTS_FLAG_POS
        return b

    @classmethod
    def decode(cls, b: int) -> CodecFlags:
        flags = cls()
        flags.eq_len = bool((b >> EQUAL_LENGTHS_FLAG_POS) & 1)
        flags.eq_tr = bool((b >> EQUAL_TIME_RANGES_FLAG_POS) & 1)
        flags.time_ranges_zero = bool((b >> TIME_RANGES_ZERO_FLAG_POS) & 1)
        flags.all_channels_present = bool((b >> ALL_CHANNELS_PRESENT_FLAG_POS) & 1)
        flags.eq_align = bool((b >> EQUAL_ALIGNMENTS_FLAG_POS) & 1)
        flags.zero_alignments = bool((b >> ZERO_ALIGNMENTS_FLAG_POS) & 1)
        return flags


class CodecState:
    keys: ChannelKeys
    data_types: Dict[ChannelKey, DataType]
    has_variable_data_types: bool

    def __init__(self, keys: ChannelKeys, data_types: List[DataType]) -> None:
        self.keys = sorted(keys)
        self.data_types = {k: dt for k, dt in zip(keys, data_types)}
        self.has_variable_data_types = any(dt.is_variable for dt in data_types)


class Codec:
    _has_variable_data_types: bool
    _seq_num: int
    _states: dict[int, CodecState]
    _curr_state: CodecState = None

    def __init__(
        self, keys: ChannelKeys = None, data_types: List[DataType] = None
    ) -> None:
        self._seq_num = 0
        self._states = dict()
        if keys is not None:
            self.update(keys, data_types)

    def update(self, keys: ChannelKeys, data_types: list[DataType]):
        self._seq_num += 1
        self._curr_state = CodecState(keys, data_types)
        self._states[self._seq_num] = self._curr_state

    def throw_if_not_updated(self, op_name: str):
        if self._curr_state is None:
            raise ValueError(
                "Codec has not been updated with keys and data types. "
                f"Please call update() before calling {op_name}()."
            )

    def encode(self, frame: Union[Frame, FramePayload], start_offset: int = 0) -> bytes:
        self.throw_if_not_updated("encode")
        pld = frame if isinstance(frame, FramePayload) else frame.to_payload()
        indices = sorted(range(len(pld.keys)), key=lambda i: pld.keys[i])
        sorted_keys = [pld.keys[i] for i in indices]
        sorted_series = [pld.series[i] for i in indices]

        flg = CodecFlags()
        flg.eq_len = not self._curr_state.has_variable_data_types
        curr_data_size = -1
        ref_tr = None
        ref_align = None
        byte_array_size = start_offset + FLAGS_SIZE + SEQ_NUM_SIZE

        if len(sorted_keys) != len(self._curr_state.keys):
            flg.all_channels_present = False
            byte_array_size += len(sorted_keys) * KEY_SIZE

        for i, ser in enumerate(sorted_series):
            key = sorted_keys[i]
            dt = self._curr_state.data_types.get(key, None)
            if dt is None:
                raise ValidationError(
                    f"encoder was provided a key {key} that is not in the codec state."
                )
            elif dt != ser.data_type:
                raise ValidationError(
                    f"data type {ser.data_type} for key {key} does not match codec state data type {dt}."
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

            byte_array_size += len(ser.data)

        flg.time_ranges_zero = ref_tr is None or ref_tr == 0
        flg.zero_alignments = flg.eq_align and ref_align == 0
        byte_array_size += (int(flg.eq_len) or len(sorted_keys)) * DATA_LENGTH_SIZE
        if not flg.time_ranges_zero:
            byte_array_size += (int(flg.eq_tr) or len(sorted_keys)) * TIME_RANGE_SIZE
        if not flg.zero_alignments:
            byte_array_size += (int(flg.eq_align) or len(sorted_keys)) * ALIGNMENT_SIZE

        buffer = bytearray(byte_array_size)
        offset = start_offset

        buffer[offset] = flg.encode()
        offset += FLAGS_SIZE

        struct.pack_into("<I", buffer, offset, self._seq_num)
        offset += SEQ_NUM_SIZE

        if flg.eq_len:
            struct.pack_into("<I", buffer, offset, curr_data_size)
            offset += DATA_LENGTH_SIZE

        if flg.eq_tr and not flg.time_ranges_zero and ref_tr:
            struct.pack_into("<QQ", buffer, offset, ref_tr.start, ref_tr.end)
            offset += TIME_RANGE_SIZE

        if flg.eq_align and not flg.zero_alignments and ref_align:
            struct.pack_into("<Q", buffer, offset, ref_align)
            offset += ALIGNMENT_SIZE

        for i, ser in enumerate(sorted_series):
            k = sorted_keys[i]
            dt = self._curr_state.data_types.get(k, None)
            if dt is None:
                raise ValidationError(
                    f"Codec state does not contain data type for key {k}."
                )
            elif dt != ser.data_type:
                raise ValidationError(
                    f"Series data type {ser.data_type} for key {k} does not match codec state data type {dt}."
                )

            if not flg.all_channels_present:
                struct.pack_into("<I", buffer, offset, sorted_keys[i])
                offset += KEY_SIZE

            if not flg.eq_len:
                len_or_size = len(ser)
                if ser.data_type.is_variable:
                    len_or_size = len(ser.data)
                struct.pack_into("<I", buffer, offset, len_or_size)
                offset += DATA_LENGTH_SIZE

            buffer[offset : offset + len(ser.data)] = ser.data
            offset += len(ser.data)

            if not flg.eq_tr and not flg.time_ranges_zero:
                if ser.time_range is None:
                    tr = TimeRange.ZERO
                else:
                    tr = ser.time_range
                struct.pack_into("<QQ", buffer, offset, tr.start, tr.end)
                offset += TIME_RANGE_SIZE

            if not flg.eq_align and not flg.zero_alignments:
                struct.pack_into("<Q", buffer, offset, ser.alignment)
                offset += ALIGNMENT_SIZE

        return bytes(buffer)

    def decode(self, data: bytes, offset: int = 0) -> FramePayload:
        self.throw_if_not_updated("decode")
        buffer = memoryview(data)
        idx = offset
        flags = CodecFlags.decode(buffer[idx])
        idx += 1

        curr_seq_num = struct.unpack_from("<I", buffer, idx)[0]
        idx += SEQ_NUM_SIZE

        state = self._states.get(curr_seq_num)
        if state is None:
            return FramePayload()

        to_del = None
        for seq_num in self._states.keys():
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
        alignment = 0

        if flags.eq_len:
            data_len = struct.unpack_from("<I", buffer, idx)[0]
            idx += DATA_LENGTH_SIZE

        if flags.eq_tr and not flags.time_ranges_zero:
            start_time, end_time = struct.unpack_from("<QQ", buffer, idx)
            idx += TIME_RANGE_SIZE

        if flags.eq_align and not flags.zero_alignments:
            alignment = struct.unpack_from("<Q", buffer, idx)[0]
            idx += ALIGNMENT_SIZE

        keys = list()
        series_list = list()

        for key in state.keys:
            if not flags.all_channels_present:
                if idx >= len(buffer):
                    break
                frame_key = struct.unpack_from("<I", buffer, idx)[0]
                if frame_key != key:
                    continue
                idx += KEY_SIZE
            data_type = state.data_types[key]
            curr_len = data_len
            if not flags.eq_len:
                curr_len = struct.unpack_from("<I", buffer, idx)[0]
                idx += DATA_LENGTH_SIZE

            data_byte_len = curr_len
            if not data_type.is_variable:
                data_byte_len = curr_len * data_type.density

            series_data = bytes(buffer[idx : idx + data_byte_len])
            idx += data_byte_len

            if flags.time_ranges_zero:
                tr = TimeRange.ZERO
            elif flags.eq_tr:
                tr = TimeRange(start=start_time, end=end_time)
            else:
                s, e = struct.unpack_from("<QQ", buffer, idx)
                tr = TimeRange(start=s, end=e)
                idx += TIME_RANGE_SIZE

            curr_alignment = alignment
            if not flags.eq_align and not flags.zero_alignments:
                curr_alignment = struct.unpack_from("<Q", buffer, idx)[0]
                idx += ALIGNMENT_SIZE

            keys.append(key)
            series_list.append(
                Series(
                    data_type=data_type,
                    data=series_data,
                    time_range=tr,
                    alignment=curr_alignment,
                )
            )

        return FramePayload(keys=keys, series=series_list)


LOW_PERF_SPECIAL_CHAR = 254
HIGH_PERF_SPECIAL_CHAR = 255
CONTENT_TYPE = "application/sy-framer"


class WSFramerCodec(FreighterCodec):
    def __init__(self, codec: Codec) -> None:
        self.codec = codec
        self.lower_perf_codec = JSONCodec()

    def content_type(self) -> str:
        return CONTENT_TYPE
