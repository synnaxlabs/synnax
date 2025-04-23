#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import numpy as np
from typing import List

from synnax.channel.payload import ChannelKeys
from synnax.framer.codec import Codec
from synnax.framer.frame import Frame
from synnax.telem import DataType, Series, TimeRange


@pytest.mark.frame_codec
class TestCodec:
    class Spec:
        def __init__(
            self,
            name: str,
            channels: ChannelKeys,
            data_types: List[DataType],
            frame: Frame
        ):
            self.name = name
            self.channels = channels
            self.data_types = data_types
            self.frame = frame

    @pytest.mark.parametrize("spec", [
        Spec(
            name="All Channels Present, In Order",
            channels=[1, 2, 3],
            data_types=[DataType.INT64, DataType.FLOAT32, DataType.FLOAT64],
            frame=Frame(
                channels=[1, 2, 3],
                series=[
                    Series(data=np.array([1, 2, 3], dtype=np.int64)),
                    Series(data=np.array([4, 5, 6], dtype=np.float32)),
                    Series(data=np.array([7, 8, 9], dtype=np.float64)),
                ],
            ),
        ),
        Spec(
            name="All Channels Present, Out of Order",
            channels=[3, 1, 2],
            data_types=[DataType.FLOAT64, DataType.INT64, DataType.FLOAT32],
            frame=Frame(
                channels=[2, 3, 1],
                series=[
                    Series(data=np.array([4, 5, 6], dtype=np.float32)),
                    Series(data=np.array([7, 8, 9], dtype=np.float64)),
                    Series(data=np.array([1, 2, 3], dtype=np.int64)),
                ],
            ),
        ),
        Spec(
            name="Some Channels Present, In Order",
            channels=[1, 2, 3],
            data_types=[DataType.UINT8, DataType.FLOAT32, DataType.FLOAT64],
            frame=Frame(
                channels=[1, 3],
                series=[
                    Series(data=np.array([1, 2, 3], dtype=np.uint8)),
                    Series(data=np.array([7, 8, 9], dtype=np.float64)),
                ],
            ),
        ),
        Spec(
            name="Some Channels Present, Out of Order",
            channels=[1, 2, 3],
            data_types=[DataType.UINT8, DataType.FLOAT32, DataType.FLOAT64],
            frame=Frame(
                channels=[3, 1],
                series=[
                    Series(data=np.array([7, 8, 9], dtype=np.float64)),
                    Series(data=np.array([1, 2, 3], dtype=np.uint8)),
                ],
            ),
        ),
        Spec(
            name="All Same Time Range",
            channels=[1, 2],
            data_types=[DataType.UINT8, DataType.FLOAT32],
            frame=Frame(
                channels=[1, 2],
                series=[
                    Series(
                        data_type=DataType.UINT8,
                        data=np.array([1], dtype=np.uint8),
                        time_range=TimeRange(start=0, end=5)
                    ),
                    Series(
                        data_type=DataType.FLOAT32,
                        data=np.array([1, 2, 3, 4], dtype=np.float32),
                        time_range=TimeRange(start=0, end=5)
                    ),
                ],
            ),
        ),
        Spec(
            name="Different Time Ranges",
            channels=[1, 2],
            data_types=[DataType.UINT8, DataType.FLOAT32],
            frame=Frame(
                channels=[1, 2],
                series=[
                    Series(
                        data_type=DataType.UINT8,
                        data=np.array([1], dtype=np.uint8),
                        time_range=TimeRange(start=0, end=5)
                    ),
                    Series(
                        data_type=DataType.FLOAT32,
                        data=np.array([1, 2, 3, 4], dtype=np.float32),
                        time_range=TimeRange(start=5, end=10)
                    ),
                ],
            ),
        ),
        Spec(
            name="Partial Present, Different Lengths",
            channels=[1, 2, 3],
            data_types=[DataType.UINT8, DataType.FLOAT32, DataType.FLOAT64],
            frame=Frame(
                channels=[1, 3],
                series=[
                    Series(data=np.array([1], dtype=np.uint8)),
                    Series(data=np.array([1, 2, 3, 4], dtype=np.float64)),
                ],
            ),
        ),
        Spec(
            name="Same Alignments",
            channels=[1, 2],
            data_types=[DataType.UINT8, DataType.FLOAT32],
            frame=Frame(
                channels=[1, 2],
                series=[
                    Series(
                        data_type=DataType.UINT8,
                        data=np.array([1], dtype=np.uint8),
                        alignment=5
                    ),
                    Series(
                        data_type=DataType.FLOAT32,
                        data=np.array([1, 2, 3, 4], dtype=np.float32),
                        alignment=5
                    ),
                ],
            ),
        ),
        Spec(
            name="Different Alignments",
            channels=[1, 2],
            data_types=[DataType.UINT8, DataType.FLOAT32],
            frame=Frame(
                channels=[1, 2],
                series=[
                    Series(
                        data_type=DataType.UINT8,
                        data=np.array([1], dtype=np.uint8),
                        alignment=5
                    ),
                    Series(
                        data_type=DataType.FLOAT32,
                        data=np.array([1, 2, 3, 4], dtype=np.float32),
                        alignment=10
                    ),
                ],
            ),
        ),
        Spec(
            name="Regression 1",
            channels=[1, 2],
            data_types=[DataType.TIMESTAMP, DataType.FLOAT32],
            frame=Frame(
                channels=[2],
                series=[
                    Series(
                        data_type=DataType.FLOAT32,
                        data=np.array([1, 2, 3, 4], dtype=np.float32),
                        time_range=TimeRange(start=0, end=5)
                    ),
                ],
            )
        ),
        Spec(
            name="Variable Data Types",
            channels=[1, 2],
            data_types=[DataType.UINT8, DataType.STRING, DataType.JSON],
            frame=Frame(
                channels=[1, 2],
                series=[
                    Series(
                        data_type=DataType.UINT8,
                        data=np.array([1], dtype=np.uint8),
                        time_range=TimeRange(start=0, end=5)
                    ),
                    Series(["cat", "dog", "orange"]),
                    Series(
                        [{"key": "value"}, {"key1": "value1"}],
                        data_type=DataType.JSON
                    ),
                ],
            )
        )
    ])
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
                assert dec_ser.time_range == TimeRange.ZERO
            else:
                assert dec_ser.time_range == or_ser.time_range
            assert dec_ser.alignment == or_ser.alignment
