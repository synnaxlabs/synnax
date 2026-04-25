#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import numpy as np
import pytest

import synnax as sy
from synnax.channel.payload import Key
from synnax.framer.codec import Codec


@pytest.mark.framer
@pytest.mark.frame_codec
class TestCodec:
    class Spec:
        def __init__(
            self,
            name: str,
            channels: list[Key] | tuple[Key],
            data_types: list[sy.DataType],
            frame: sy.Frame,
        ):
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
                channels=[1, 2],
                data_types=[sy.DataType.UINT8, sy.DataType.STRING, sy.DataType.JSON],
                frame=sy.Frame(
                    channels=[1, 2],
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
                data_types=[sy.DataType.BOOL],
                frame=sy.Frame(
                    channels=[1],
                    series=[sy.Series([True], data_type=sy.DataType.BOOL)],
                ),
            ),
            Spec(
                name="Bool Exact Byte Boundary",
                channels=[1],
                data_types=[sy.DataType.BOOL],
                frame=sy.Frame(
                    channels=[1],
                    series=[
                        sy.Series(
                            [True, False, True, False, True, False, True, False],
                            data_type=sy.DataType.BOOL,
                        )
                    ],
                ),
            ),
            Spec(
                name="Bool One Past Byte Boundary",
                channels=[1],
                data_types=[sy.DataType.BOOL],
                frame=sy.Frame(
                    channels=[1],
                    series=[
                        sy.Series(
                            [True, False, True, False, True, False, True, False, True],
                            data_type=sy.DataType.BOOL,
                        )
                    ],
                ),
            ),
            Spec(
                name="Bool Seven Samples Partial Last Byte",
                channels=[1],
                data_types=[sy.DataType.BOOL],
                frame=sy.Frame(
                    channels=[1],
                    series=[
                        sy.Series(
                            [True, False, True, True, False, False, True],
                            data_type=sy.DataType.BOOL,
                        )
                    ],
                ),
            ),
            Spec(
                name="Bool Mixed With Other Types",
                channels=[1, 2, 3],
                data_types=[
                    sy.DataType.BOOL,
                    sy.DataType.FLOAT32,
                    sy.DataType.UINT8,
                ],
                frame=sy.Frame(
                    channels=[1, 2, 3],
                    series=[
                        sy.Series(
                            [True, False, True],
                            data_type=sy.DataType.BOOL,
                        ),
                        sy.Series(np.array([1.5, 2.5, 3.5], dtype=np.float32)),
                        sy.Series(np.array([7, 8, 9], dtype=np.uint8)),
                    ],
                ),
            ),
        ],
    )
    def test_encoder_decoder(self, spec: Spec):
        codec = Codec(spec.channels, spec.data_types)
        encoded = codec.encode(spec.frame)
        decoded = codec.decode(encoded)
        assert len(decoded.keys) == len(spec.frame.channels)
        for i, key in enumerate(decoded.keys):
            dec_ser = decoded.series[i]
            or_ser = spec.frame[key].series[0]
            assert np.array_equal(list(dec_ser), list(or_ser))
            if or_ser.time_range is None:
                assert dec_ser.time_range == sy.TimeRange.ZERO
            else:
                assert dec_ser.time_range == or_ser.time_range
            assert dec_ser.alignment == or_ser.alignment

    def test_bool_reference_vector(self):
        """The reference vector [1,0,1,1,0,0,0,1,1] must pack to bytes [0x8D, 0x01]
        LSB-first across all language codecs. Any divergence here is a cross-language
        wire-format bug.
        """
        from synnax.framer.codec import _pack_bool_bits, _unpack_bool_bits

        samples = bytes([1, 0, 1, 1, 0, 0, 0, 1, 1])
        packed = _pack_bool_bits(samples)
        assert packed == bytes([0x8D, 0x01])
        unpacked = _unpack_bool_bits(memoryview(packed), len(samples))
        assert unpacked == samples

    def test_dynamic_codec_update(self):
        """Tests that the codec can be updated with new channels and correctly encode/decode after update"""
        codec = Codec()
        # First update and verification
        codec.update([1], [sy.DataType.INT32])
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))]
        )
        encoded = codec.encode(frame)
        decoded = codec.decode(encoded)
        assert len(decoded.keys) == 1
        assert decoded.keys[0] == 1
        assert np.array_equal(list(decoded.series[0]), [1, 2, 3])

        # Second update and verification
        codec.update([2], [sy.DataType.INT64])
        frame2 = sy.Frame(
            channels=[2], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int64))]
        )
        encoded2 = codec.encode(frame2)
        decoded2 = codec.decode(encoded2)
        assert len(decoded2.keys) == 1
        assert decoded2.keys[0] == 2
        assert np.array_equal(list(decoded2.series[0]), [1, 2, 3])

    def test_uninitialized_codec(self):
        """Tests that using an uninitialized codec raises appropriate errors"""
        codec = Codec()
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))]
        )
        with pytest.raises(ValueError):
            codec.encode(frame)

    def test_out_of_sync_codecs(self):
        """Tests correct behavior when encoder and decoder are out of sync with different states"""
        encoder = Codec()
        decoder = Codec()

        # Initial state - both in sync
        encoder.update([1], [sy.DataType.INT32])
        decoder.update([1], [sy.DataType.INT32])
        frame = sy.Frame(
            channels=[1], series=[sy.Series(data=np.array([1, 2, 3], dtype=np.int32))]
        )
        encoded = encoder.encode(frame)
        decoded = decoder.decode(encoded)
        assert len(decoded.keys) == 1
        assert decoded.keys[0] == 1
        assert np.array_equal(list(decoded.series[0]), [1, 2, 3])

        # Decoder updates but encoder doesn't - should still work with old format
        decoder.update([2], [sy.DataType.INT64])
        encoded = encoder.encode(frame)
        decoded = decoder.decode(encoded)
        assert len(decoded.keys) == 1
        assert decoded.keys[0] == 1
        assert np.array_equal(list(decoded.series[0]), [1, 2, 3])

        # Encoder updates - old frame should now fail
        encoder.update([2], [sy.DataType.INT64])
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
        assert np.array_equal(list(decoded2.series[0]), [1, 2, 3])
