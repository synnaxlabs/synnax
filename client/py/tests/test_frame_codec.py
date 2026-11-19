#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import json as json_lib
import pathlib
import re
import struct
from collections.abc import Callable
from typing import Any

import numpy as np
import pytest

import synnax as sy
from freighter import ExceptionPayload
from freighter.websocket import Message
from synnax.channel.payload import Key
from synnax.framer.codec import (
    _EXTENDED_FLAG_POS,
    _MULTIPLES_PRESENT_FLAG_POS,
    LOW_PERF_SPECIAL_CHAR,
    Codec,
    _pack_bool_bits,
    _unpack_bool_bits,
)
from synnax.framer.frame import FramePayload
from synnax.framer.iterator import (
    _Command,
    _Response,
    _ResponseVariant,
    _WSIteratorCodec,
)
from synnax.framer.writer import WriterCommand, WriterResponse, _WSWriterCodec


@pytest.mark.framer
@pytest.mark.frame_codec
class TestBoolBits:
    def test_reference_vector(self) -> None:
        """Should pack LSB first to the same bytes as the Go and TypeScript codecs."""
        samples = bytes([1, 0, 1, 1, 0, 0, 0, 1, 1])
        packed = _pack_bool_bits(samples)
        assert packed == bytes([0x8D, 0x01])
        assert _unpack_bool_bits(memoryview(packed), len(samples)) == samples

    def test_nonzero_packs_true(self) -> None:
        """Should pack every nonzero byte as a set bit."""
        packed = _pack_bool_bits(bytes([2, 0, 255, 0, 0, 0, 0, 0, 7]))
        assert packed == bytes([0x05, 0x01])
        assert _unpack_bool_bits(memoryview(packed), 9) == bytes(
            [1, 0, 1, 0, 0, 0, 0, 0, 1]
        )

    @pytest.mark.parametrize("n", [0, 1, 7, 8, 9, 15, 16, 17, 1000, 1003])
    def test_round_trip(self, n: int) -> None:
        """Should round trip any sample count, including partial final bytes."""
        rng = np.random.default_rng(n)
        samples = rng.integers(0, 2, n, dtype=np.uint8).tobytes()
        packed = _pack_bool_bits(samples)
        assert len(packed) == (n + 7) // 8
        assert _unpack_bool_bits(memoryview(packed), n) == samples

    def test_unpack_ignores_padding_bits(self) -> None:
        """Should ignore set bits past the sample count in the final byte."""
        assert _unpack_bool_bits(memoryview(bytes([0xFF])), 3) == bytes([1, 1, 1])


@pytest.mark.framer
@pytest.mark.frame_codec
class TestCodec:
    class Spec:
        def __init__(
            self,
            name: str,
            channels: list[Key],
            data_types: list[sy.DataType],
            frame: sy.Frame,
        ) -> None:
            self.name = name
            self.channels = channels
            self.data_types = data_types
            self.frame = frame

    @pytest.mark.parametrize(
        "spec",
        [
            Spec(
                name="All Channels Present, In Order",
                channels=[1, 2, 3],
                data_types=[
                    sy.DataType.INT64,
                    sy.DataType.FLOAT32,
                    sy.DataType.FLOAT64,
                ],
                frame=sy.Frame(
                    channels=[1, 2, 3],
                    series=[
                        sy.Series(data=np.array([1, 2, 3], dtype=np.int64)),
                        sy.Series(data=np.array([4, 5, 6], dtype=np.float32)),
                        sy.Series(data=np.array([7, 8, 9], dtype=np.float64)),
                    ],
                ),
            ),
            Spec(
                name="All Channels Present, Out of Order",
                channels=[3, 1, 2],
                data_types=[
                    sy.DataType.FLOAT64,
                    sy.DataType.INT64,
                    sy.DataType.FLOAT32,
                ],
                frame=sy.Frame(
                    channels=[2, 3, 1],
                    series=[
                        sy.Series(data=np.array([4, 5, 6], dtype=np.float32)),
                        sy.Series(data=np.array([7, 8, 9], dtype=np.float64)),
                        sy.Series(data=np.array([1, 2, 3], dtype=np.int64)),
                    ],
                ),
            ),
            Spec(
                name="Some Channels Present, In Order",
                channels=[1, 2, 3],
                data_types=[
                    sy.DataType.UINT8,
                    sy.DataType.FLOAT32,
                    sy.DataType.FLOAT64,
                ],
                frame=sy.Frame(
                    channels=[1, 3],
                    series=[
                        sy.Series(data=np.array([1, 2, 3], dtype=np.uint8)),
                        sy.Series(data=np.array([7, 8, 9], dtype=np.float64)),
                    ],
                ),
            ),
            Spec(
                name="Some Channels Present, Out of Order",
                channels=[1, 2, 3],
                data_types=[
                    sy.DataType.UINT8,
                    sy.DataType.FLOAT32,
                    sy.DataType.FLOAT64,
                ],
                frame=sy.Frame(
                    channels=[3, 1],
                    series=[
                        sy.Series(data=np.array([7, 8, 9], dtype=np.float64)),
                        sy.Series(data=np.array([1, 2, 3], dtype=np.uint8)),
                    ],
                ),
            ),
            Spec(
                name="Only One Channel Present",
                channels=[1, 2, 3, 4, 5],
                data_types=[
                    sy.DataType.UINT8,
                    sy.DataType.UINT8,
                    sy.DataType.UINT8,
                    sy.DataType.UINT8,
                    sy.DataType.UINT8,
                ],
                frame=sy.Frame(
                    channels=[3],
                    series=[
                        sy.Series(data=np.array([1, 2, 3, 4, 5], dtype=np.uint8)),
                    ],
                ),
            ),
            Spec(
                name="All Same Time Range",
                channels=[1, 2],
                data_types=[sy.DataType.UINT8, sy.DataType.FLOAT32],
                frame=sy.Frame(
                    channels=[1, 2],
                    series=[
                        sy.Series(
                            data_type=sy.DataType.UINT8,
                            data=np.array([1], dtype=np.uint8),
                            time_range=sy.TimeRange(start=0, end=5),
                        ),
                        sy.Series(
                            data_type=sy.DataType.FLOAT32,
                            data=np.array([1, 2, 3, 4], dtype=np.float32),
                            time_range=sy.TimeRange(start=0, end=5),
                        ),
                    ],
                ),
            ),
            Spec(
                name="Different Time Ranges",
                channels=[1, 2],
                data_types=[sy.DataType.UINT8, sy.DataType.FLOAT32],
                frame=sy.Frame(
                    channels=[1, 2],
                    series=[
                        sy.Series(
                            data_type=sy.DataType.UINT8,
                            data=np.array([1], dtype=np.uint8),
                            time_range=sy.TimeRange(start=0, end=5),
                        ),
                        sy.Series(
                            data_type=sy.DataType.FLOAT32,
                            data=np.array([1, 2, 3, 4], dtype=np.float32),
                            time_range=sy.TimeRange(start=5, end=10),
                        ),
                    ],
                ),
            ),
            Spec(
                name="Partial Present, Different Lengths",
                channels=[1, 2, 3],
                data_types=[
                    sy.DataType.UINT8,
                    sy.DataType.FLOAT32,
                    sy.DataType.FLOAT64,
                ],
                frame=sy.Frame(
                    channels=[1, 3],
                    series=[
                        sy.Series(data=np.array([1], dtype=np.uint8)),
                        sy.Series(data=np.array([1, 2, 3, 4], dtype=np.float64)),
                    ],
                ),
            ),
            Spec(
                name="Same Alignments",
                channels=[1, 2],
                data_types=[sy.DataType.UINT8, sy.DataType.FLOAT32],
                frame=sy.Frame(
                    channels=[1, 2],
                    series=[
                        sy.Series(
                            data_type=sy.DataType.UINT8,
                            data=np.array([1], dtype=np.uint8),
                            alignment=5,
                        ),
                        sy.Series(
                            data_type=sy.DataType.FLOAT32,
                            data=np.array([1, 2, 3, 4], dtype=np.float32),
                            alignment=5,
                        ),
                    ],
                ),
            ),
            Spec(
                name="Different Alignments",
                channels=[1, 2],
                data_types=[sy.DataType.UINT8, sy.DataType.FLOAT32],
                frame=sy.Frame(
                    channels=[1, 2],
                    series=[
                        sy.Series(
                            data_type=sy.DataType.UINT8,
                            data=np.array([1], dtype=np.uint8),
                            alignment=5,
                        ),
                        sy.Series(
                            data_type=sy.DataType.FLOAT32,
                            data=np.array([1, 2, 3, 4], dtype=np.float32),
                            alignment=10,
                        ),
                    ],
                ),
            ),
            Spec(
                name="Regression 1",
                channels=[1, 2],
                data_types=[sy.DataType.TIMESTAMP, sy.DataType.FLOAT32],
                frame=sy.Frame(
                    channels=[2],
                    series=[
                        sy.Series(
                            data_type=sy.DataType.FLOAT32,
                            data=np.array([1, 2, 3, 4], dtype=np.float32),
                            time_range=sy.TimeRange(start=0, end=5),
                        ),
                    ],
                ),
            ),
            Spec(
                name="Variable Data Types",
                channels=[1, 2, 3],
                data_types=[sy.DataType.UINT8, sy.DataType.STRING, sy.DataType.JSON],
                frame=sy.Frame(
                    channels=[1, 2, 3],
                    series=[
                        sy.Series(
                            data_type=sy.DataType.UINT8,
                            data=np.array([1], dtype=np.uint8),
                            time_range=sy.TimeRange(start=0, end=5),
                        ),
                        sy.Series(["cat", "dog", "orange"]),
                        sy.Series(
                            [{"key": "value"}, {"key1": "value1"}],
                            data_type=sy.DataType.JSON,
                        ),
                    ],
                ),
            ),
            Spec(
                name="Bool Single Sample",
                channels=[1],
                data_types=[sy.DataType.BOOLEAN],
                frame=sy.Frame(
                    channels=[1],
                    series=[sy.Series([True], data_type=sy.DataType.BOOLEAN)],
                ),
            ),
            Spec(
                name="Bool Exact Byte Boundary",
                channels=[1],
                data_types=[sy.DataType.BOOLEAN],
                frame=sy.Frame(
                    channels=[1],
                    series=[
                        sy.Series(
                            [True, False, True, False, True, False, True, False],
                            data_type=sy.DataType.BOOLEAN,
                        )
                    ],
                ),
            ),
            Spec(
                name="Bool One Past Byte Boundary",
                channels=[1],
                data_types=[sy.DataType.BOOLEAN],
                frame=sy.Frame(
                    channels=[1],
                    series=[
                        sy.Series(
                            [True, False, True, False, True, False, True, False, True],
                            data_type=sy.DataType.BOOLEAN,
                        )
                    ],
                ),
            ),
            Spec(
                name="Bool Seven Samples Partial Last Byte",
                channels=[1],
                data_types=[sy.DataType.BOOLEAN],
                frame=sy.Frame(
                    channels=[1],
                    series=[
                        sy.Series(
                            [True, False, True, True, False, False, True],
                            data_type=sy.DataType.BOOLEAN,
                        )
                    ],
                ),
            ),
            Spec(
                name="Bool Mixed With Other Types",
                channels=[1, 2, 3],
                data_types=[
                    sy.DataType.BOOLEAN,
                    sy.DataType.FLOAT32,
                    sy.DataType.UINT8,
                ],
                frame=sy.Frame(
                    channels=[1, 2, 3],
                    series=[
                        sy.Series(
                            [True, False, True],
                            data_type=sy.DataType.BOOLEAN,
                        ),
                        sy.Series(np.array([1.5, 2.5, 3.5], dtype=np.float32)),
                        sy.Series(np.array([7, 8, 9], dtype=np.uint8)),
                    ],
                ),
            ),
        ],
    )
    def test_encoder_decoder(self, spec: Spec) -> None:
        codec = Codec(keys=spec.channels, data_types=spec.data_types)
        encoded = codec.encode(spec.frame)
        decoded = codec.decode(encoded)
        assert len(decoded.keys) == len(spec.frame.channels)
        for i, key in enumerate(decoded.keys):
            dec_ser = decoded.series[i]
            or_ser = spec.frame[key].series[0]
            assert list(dec_ser) == list(or_ser)
            if or_ser.time_range is None:
                assert dec_ser.time_range == sy.TimeRange.ZERO
            else:
                assert dec_ser.time_range == or_ser.time_range
            assert dec_ser.alignment == or_ser.alignment

    def test_alignment_multiples_round_trip(self) -> None:
        """Should carry the alignment multiple of every series when any exceeds one."""
        codec = Codec(keys=[1, 2], data_types=[sy.DataType.INT64, sy.DataType.FLOAT32])
        frame = sy.Frame(
            channels=[1, 2],
            series=[
                sy.Series(
                    data=np.array([1, 2], dtype=np.int64),
                    alignment=10,
                    alignment_multiple=4,
                ),
                sy.Series(data=np.array([3, 4], dtype=np.float32), alignment=20),
            ],
        )
        encoded = codec.encode(frame)
        assert (encoded[0] >> _MULTIPLES_PRESENT_FLAG_POS) & 1
        decoded = codec.decode(encoded)
        assert [s.alignment_multiple for s in decoded.series] == [4, 1]
        assert [int(s.alignment) for s in decoded.series] == [10, 20]
        assert np.array_equal(decoded.series[0].to_numpy(), [1, 2])
        assert np.array_equal(decoded.series[1].to_numpy(), [3, 4])

    def test_time_range_after_series_without_one(self) -> None:
        """Should keep the time range of a series after a series without one."""
        codec = Codec(keys=[1, 2], data_types=[sy.DataType.INT64, sy.DataType.INT64])
        frame = sy.Frame(
            channels=[1, 2],
            series=[
                sy.Series(data=np.array([1, 2], dtype=np.int64)),
                sy.Series(
                    data=np.array([3, 4], dtype=np.int64),
                    time_range=sy.TimeRange(start=5, end=10),
                ),
            ],
        )
        decoded = codec.decode(codec.encode(frame))
        assert decoded.series[0].time_range == sy.TimeRange.ZERO
        assert decoded.series[1].time_range == sy.TimeRange(start=5, end=10)

    def test_no_alignment_multiples(self) -> None:
        """Should leave the multiples flag clear when no series is reduced."""
        codec = Codec(keys=[1], data_types=[sy.DataType.INT64])
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2], dtype=np.int64))]
        )
        encoded = codec.encode(frame)
        assert not (encoded[0] >> _MULTIPLES_PRESENT_FLAG_POS) & 1
        assert codec.decode(encoded).series[0].alignment_multiple == 1

    def test_first_frame_sets_extended_flag(self) -> None:
        """Should set the extended bit, which shipped decoders reject, in the first
        frame of a state."""
        codec = Codec(keys=[1], data_types=[sy.DataType.INT64])
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2], dtype=np.int64))]
        )
        assert (codec.encode(frame)[0] >> _EXTENDED_FLAG_POS) & 1

    def test_rejects_frame_older_than_last_decoded(self) -> None:
        """Should reject a frame encoded with keys older than the last frame decoded."""
        encoder = Codec(keys=[1], data_types=[sy.DataType.INT32])
        decoder = Codec(keys=[1], data_types=[sy.DataType.INT32])
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))]
        )
        stale = encoder.encode(frame)
        encoder.update(keys=[1, 2], data_types=[sy.DataType.INT32, sy.DataType.INT64])
        decoder.update(keys=[1, 2], data_types=[sy.DataType.INT32, sy.DataType.INT64])
        decoder.decode(encoder.encode(frame))
        with pytest.raises(
            sy.ValidationError, match=r"^remote sent invalid sequence number 1$"
        ):
            decoder.decode(stale)

    def test_dynamic_codec_update(self) -> None:
        """Tests that the codec can be updated with new channels and correctly
        encode/decode after update
        """
        codec = Codec()
        # First update and verification
        codec.update(keys=[1], data_types=[sy.DataType.INT32])
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))]
        )
        encoded = codec.encode(frame)
        decoded = codec.decode(encoded)
        assert len(decoded.keys) == 1
        assert decoded.keys[0] == 1
        assert list(decoded.series[0]) == [1, 2, 3]

        # Second update and verification
        codec.update(keys=[2], data_types=[sy.DataType.INT64])
        frame2 = sy.Frame(
            channels=[2], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int64))]
        )
        encoded2 = codec.encode(frame2)
        decoded2 = codec.decode(encoded2)
        assert len(decoded2.keys) == 1
        assert decoded2.keys[0] == 2
        assert list(decoded2.series[0]) == [1, 2, 3]

    def test_uninitialized_codec(self) -> None:
        """Tests that using an uninitialized codec raises appropriate errors"""
        codec = Codec()
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))]
        )
        with pytest.raises(ValueError, match="Codec has not been updated"):
            codec.encode(frame)

    def test_iterator_ws_codec_data_response(self) -> None:
        """Should binary-decode a data variant response and synthesize variant=Data"""
        base = Codec(keys=[1], data_types=[sy.DataType.INT32])
        codec = _WSIteratorCodec(base)
        frame = sy.Frame(
            channels=[1],
            series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))],
        )
        encoded = bytearray(base.encode(frame, start_offset=1))
        encoded[0] = 0xFF
        msg = codec.decode(bytes(encoded), Message[_Response])
        assert msg.payload is not None
        assert msg.payload.variant == _ResponseVariant.DATA
        assert msg.payload.frame.keys == [1]
        assert list(msg.payload.frame.series[0]) == [1, 2, 3]

    def test_iterator_ws_codec_ack_response(self) -> None:
        """Should JSON-decode an ack variant response prefixed with the low-perf char"""
        base = Codec(keys=[1], data_types=[sy.DataType.INT32])
        codec = _WSIteratorCodec(base)
        ack = Message[_Response](
            type="data",
            payload=_Response(
                variant=_ResponseVariant.ACK,
                command=_Command.NEXT,
                ack=True,
                error=None,
                frame=FramePayload(),
            ),
        )
        body = json_lib.dumps(ack.model_dump(mode="json", by_alias=True)).encode()
        encoded = bytes([LOW_PERF_SPECIAL_CHAR]) + body
        decoded = codec.decode(encoded, Message[_Response])
        assert decoded.payload is not None
        assert decoded.payload.variant == _ResponseVariant.ACK
        assert decoded.payload.ack is True

    def test_iterator_ws_codec_request_is_json(self) -> None:
        """Should JSON-encode an iterator request without a special-char prefix"""
        base = Codec(keys=[1], data_types=[sy.DataType.INT32])
        codec = _WSIteratorCodec(base)

        from synnax.framer.iterator import _Request

        req = Message[_Request](
            type="data",
            payload=_Request(command=_Command.NEXT),
        )
        encoded = codec.encode(req)
        assert encoded[0] in (ord("{"), ord("["))

    def test_out_of_sync_codecs(self) -> None:
        """Tests correct behavior when encoder and decoder are out of sync with
        different states
        """
        encoder = Codec()
        decoder = Codec()

        # Initial state - both in sync
        encoder.update(keys=[1], data_types=[sy.DataType.INT32])
        decoder.update(keys=[1], data_types=[sy.DataType.INT32])
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))]
        )
        encoded = encoder.encode(frame)
        decoded = decoder.decode(encoded)
        assert len(decoded.keys) == 1
        assert decoded.keys[0] == 1
        assert list(decoded.series[0]) == [1, 2, 3]

        # Decoder updates but encoder doesn't - should still work with old format
        decoder.update(keys=[2], data_types=[sy.DataType.INT64])
        encoded = encoder.encode(frame)
        decoded = decoder.decode(encoded)
        assert len(decoded.keys) == 1
        assert decoded.keys[0] == 1
        assert list(decoded.series[0]) == [1, 2, 3]

        # Encoder updates - old frame should now fail
        encoder.update(keys=[2], data_types=[sy.DataType.INT64])
        with pytest.raises(sy.ValidationError):
            encoder.encode(frame)

        # New frame with updated channel should work
        frame2 = sy.Frame(
            channels=[2], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int64))]
        )
        encoded2 = encoder.encode(frame2)
        decoded2 = decoder.decode(encoded2)
        assert len(decoded2.keys) == 1
        assert decoded2.keys[0] == 2
        assert list(decoded2.series[0]) == [1, 2, 3]

    def test_writer_ws_codec_response(self) -> None:
        """Should JSON-decode a writer response prefixed with the low-perf char"""
        codec = _WSWriterCodec(Codec(keys=[1], data_types=[sy.DataType.INT32]))
        res = Message[WriterResponse](
            type="data",
            payload=WriterResponse(
                command=WriterCommand.COMMIT,
                end=sy.TimeStamp(5),
                err=ExceptionPayload(type=None, data=None),
                authorized=True,
            ),
        )
        body = json_lib.dumps(res.model_dump(mode="json", by_alias=True)).encode()
        decoded = codec.decode(
            bytes([LOW_PERF_SPECIAL_CHAR]) + body, Message[WriterResponse]
        )
        assert decoded.payload is not None
        assert decoded.payload.command == WriterCommand.COMMIT
        assert decoded.payload.end == sy.TimeStamp(5)

    def test_writer_ws_codec_rejects_data_frame(self) -> None:
        """Should fail on a high-performance data frame, which the Core never sends
        on the writer response stream"""
        base = Codec(keys=[1], data_types=[sy.DataType.INT32])
        codec = _WSWriterCodec(base)
        frame = sy.Frame(
            channels=[1],
            series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))],
        )
        encoded = bytearray(base.encode(frame, start_offset=1))
        encoded[0] = 0xFF
        with pytest.raises(ValueError, match="writer response stream"):
            codec.decode(bytes(encoded), Message[WriterResponse])


_VECTORS_PATH = (
    pathlib.Path(__file__).parents[3] / "x" / "testdata" / "framecodec" / "vectors.json"
)
"""The golden frames that the Go, TypeScript, and C++ codecs share."""

_VECTOR_CASES = json_lib.loads(_VECTORS_PATH.read_text())["cases"]


def _hex(h: str) -> bytes:
    return bytes.fromhex(h.replace(" ", ""))


class TestGoldenFrames:
    @pytest.mark.parametrize(
        "case", _VECTOR_CASES, ids=[c["name"] for c in _VECTOR_CASES]
    )
    def test_byte_for_byte(self, case: dict[str, Any]) -> None:
        """Should encode and decode each golden frame byte for byte."""
        data_types = [sy.DataType(dt) for dt in case["data_types"]]
        enc = Codec(keys=case["keys"], data_types=data_types)
        dec = Codec(keys=case["keys"], data_types=data_types)
        for f in case["frames"]:
            pld = FramePayload(
                keys=[s["key"] for s in f["series"]],
                series=[
                    sy.Series(
                        data=bytes.fromhex(s["data"]),
                        data_type=s["data_type"],
                        clock=s.get("clock", 0),
                    )
                    for s in f["series"]
                ],
            )
            assert enc.encode(pld).hex() == f["encoded"], f["name"]
            decoded = dec.decode(bytes.fromhex(f["encoded"]))
            assert decoded.keys == [s["key"] for s in f["series"]], f["name"]
            for got, want in zip(decoded.series, f["series"], strict=True):
                assert got.data_type == want["data_type"], f["name"]
                assert got.clock == want.get("clock", 0), f["name"]
                assert got.data.hex() == want["data"], f["name"]


# The first frame of a float32 key 1 whose one sample, 1.0, has clock 5. Byte 5 is the
# extension flags, byte 6 the type code, byte 15 the override kinds, and bytes 20 to 23
# the next override ordinal.
_CLOCK_FRAME = "BF 01000000 03 09 01000000 00000000 02 05000000 FFFFFFFF 0000803F"

_WORKED_EXAMPLE = (
    "BF 01000000 0B 0901 01000000 05000000 02000000 01000000"
    " E803000000000000 D007000000000000"
    " 03 0A 05000000 FFFFFFFF 000000000000F03F 0000000000000040 0100"
)


def _float32_codec() -> Codec:
    return Codec(keys=[1], data_types=[sy.DataType.FLOAT32])


def _set(i: int, v: int) -> Callable[[bytearray], bytearray]:
    def corrupt(b: bytearray) -> bytearray:
        b[i] = v
        return b

    return corrupt


def _or(i: int, v: int) -> Callable[[bytearray], bytearray]:
    def corrupt(b: bytearray) -> bytearray:
        b[i] |= v
        return b

    return corrupt


def _u32(i: int, v: int) -> Callable[[bytearray], bytearray]:
    def corrupt(b: bytearray) -> bytearray:
        struct.pack_into("<I", b, i, v)
        return b

    return corrupt


class TestAgreement:
    def test_int64_on_timestamp_key(self) -> None:
        """Should send an int64 series on a timestamp key with no override, and label
        it with the agreed type."""
        enc = Codec(keys=[1], data_types=[sy.DataType.TIMESTAMP])
        pld = FramePayload(keys=[1], series=[sy.Series(np.array([7], dtype=np.int64))])
        encoded = enc.encode(pld)
        assert encoded == _hex("BF 01000000 01 0B 01000000 0700000000000000")
        dec = Codec(keys=[1], data_types=[sy.DataType.TIMESTAMP])
        assert dec.decode(encoded).series[0].data_type == sy.DataType.TIMESTAMP

    def test_key_named_twice(self) -> None:
        """Should count a key named twice once."""
        f32 = sy.DataType.FLOAT32
        enc = Codec(keys=[1, 1], data_types=[f32, f32])
        pld = FramePayload(keys=[1], series=[sy.Series(np.array([1], np.float32))])
        assert enc.encode(pld) == _hex("BF 01000000 01 09 01000000 0000803F")

    def test_type_override(self) -> None:
        """Should carry a type override for its own series only."""
        enc, dec = _float32_codec(), _float32_codec()
        for arr in (np.array([1], np.float64), np.array([2], np.float32)):
            pld = FramePayload(keys=[1], series=[sy.Series(arr)])
            got = dec.decode(enc.encode(pld)).series[0]
            assert got.data_type == sy.DataType(arr.dtype)

    def test_commits_nothing_on_error(self) -> None:
        """Should commit nothing when a frame fails to encode."""
        c = _float32_codec()
        bad = sy.Series(data=b"\x00" * 4, data_type=sy.DataType.FLOAT32, clock=5)
        bad.data_type = sy.DataType.UNKNOWN
        with pytest.raises(
            sy.ValidationError, match=r'^data type "" of channel 1 has no type code$'
        ):
            c.encode(FramePayload(keys=[1], series=[bad]))
        good = sy.Series(np.array([1], np.float32), clock=5)
        assert c.encode(FramePayload(keys=[1], series=[good])) == _hex(_CLOCK_FRAME)

    @pytest.mark.parametrize(
        ("corrupt", "msg"),
        [
            (
                _or(5, 0b100),
                "remote sent extension flags 00000111, which hold a reserved bit",
            ),
            (
                _or(5, 0b1000_0000),
                "remote sent extension flags 10000011, which hold a reserved bit",
            ),
            (_set(5, 0), "remote sent extension flags of 0"),
            (
                _set(6, 0),
                "remote sent data type code 0, which this codec does not know",
            ),
            (
                _set(6, 17),
                "remote sent data type code 17, which this codec does not know",
            ),
            (
                _set(6, 13),
                "remote sent variable data type string under a shared length",
            ),
            (_set(15, 0), "remote sent override kinds of 0 for series 0"),
            (
                _or(15, 0b100),
                "remote sent override kinds 00000110 for series 0, which hold a "
                "reserved bit",
            ),
            (_u32(20, 0), "remote sent next override 0, which is not after series 0"),
            (
                _u32(20, 1),
                "remote ended the frame with the override for series 1 pending",
            ),
            (lambda b: b + b"\x00", "remote sent bytes after the last series"),
        ],
    )
    def test_rejects(self, corrupt: Callable[[bytearray], bytearray], msg: str) -> None:
        """Should reject a malformed frame and drop its state."""
        dec = _float32_codec()
        with pytest.raises(sy.ValidationError, match=f"^{re.escape(msg)}$"):
            dec.decode(bytes(corrupt(bytearray(_hex(_CLOCK_FRAME)))))
        with pytest.raises(
            sy.ValidationError,
            match=(
                r"^remote sent a frame for state 1 after a frame of it failed to "
                r"decode$"
            ),
        ):
            dec.decode(_hex(_CLOCK_FRAME))

    def test_rejects_second_declaration(self) -> None:
        """Should reject a second type declaration for a state."""
        dec = _float32_codec()
        dec.decode(_hex(_CLOCK_FRAME))
        with pytest.raises(
            sy.ValidationError, match=r"^remote declared the types of state 1 twice$"
        ):
            dec.decode(_hex(_CLOCK_FRAME))

    def test_names_older_codec(self) -> None:
        """Should name the older codec of a remote that declares no types."""
        with pytest.raises(
            sy.ValidationError,
            match=(
                r"^remote sent no type declaration for state 1, so it runs an older "
                r"codec, which this codec does not support$"
            ),
        ):
            _float32_codec().decode(_hex("3F 01000000 01000000 0000803F"))

    @pytest.mark.parametrize(
        ("corrupt", "msg"),
        [
            (_u32(12, 6), "remote declared clock 6, which the request names"),
            (
                lambda b: _u32(20, 0)(b)[:24] + bytearray([0b10]) + b[24:],
                "remote sent an override for declared clock 5",
            ),
            (_u32(8, 0xFFFFFFFF), "remote sent a frame that ends early"),
        ],
    )
    def test_rejects_declared_clocks(
        self, corrupt: Callable[[bytearray], bytearray], msg: str
    ) -> None:
        """Should reject a malformed list of declared clocks."""
        dec = Codec(keys=[6, 8], data_types=[sy.DataType.FLOAT32, sy.DataType.UINT8])
        with pytest.raises(sy.ValidationError, match=f"^{re.escape(msg)}$"):
            dec.decode(bytes(corrupt(bytearray(_hex(_WORKED_EXAMPLE)))))

    def test_rejects_unordered_list(self) -> None:
        """Should reject a list of declared clocks that is not ascending."""
        c = Codec(keys=[6], data_types=[sy.DataType.FLOAT32])
        ts = sy.DataType.TIMESTAMP
        pld = FramePayload(
            keys=[5, 6, 7],
            series=[
                sy.Series(np.array([1], np.int64), data_type=ts, clock=5),
                sy.Series(np.array([1], np.float32)),
                sy.Series(np.array([1], np.int64), data_type=ts, clock=7),
            ],
        )
        encoded = bytearray(c.encode(pld))
        assert struct.unpack_from("<I", encoded, 7)[0] == 2
        struct.pack_into("<II", encoded, 11, 7, 5)
        with pytest.raises(
            sy.ValidationError,
            match=(
                r"^remote declared clock 5 after clock 7, so the list is not "
                r"ascending$"
            ),
        ):
            c.decode(bytes(encoded))
