from dataclasses import asdict

import numpy as np
import pytest
from freighter import encoder

from synnax import telem
from synnax.channel import Channel
from synnax.segment import BinarySegment, NumpySegment, SugaredBinarySegment
from synnax.segment.splitter import Splitter
from synnax.segment.validate import ScalarType, Contiguity
from synnax import errors


class TestBinarySegment:
    @pytest.mark.parametrize("ecd", [encoder.Msgpack])
    def test_encode_decode(self, ecd: encoder.EncoderDecoder):
        segment = BinarySegment(
            channel_key="1-1",
            start=telem.TimeStamp(1),
            data=b"12345",
        )
        encoded = ecd.encode(segment)
        decoded = BinarySegment()
        ecd.decode(encoded, decoded)
        assert asdict(segment) == asdict(decoded)


class TestScalarTypeValidator:
    def test_valid_segment(self):
        """
        Should not raise a validation error
        """
        ch = Channel(data_type=telem.INT64)
        seg = NumpySegment(
            channel=ch,
            start=telem.now(),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        try:
            ScalarType().validate(seg)
        except Exception as e:
            pytest.fail(f"Unexpected exception: {e}")

    def test_invalid_segment(self):
        """
        Should raise a validation error
        """
        ch = Channel(data_type=telem.INT64)
        seg = NumpySegment(
            channel=ch,
            start=telem.now(),
            data=np.array([1, 2, 3], dtype=np.int32),
        )
        with pytest.raises(errors.ValidationError):
            ScalarType().validate(seg)

    def test_unrecognized_data_type(self):
        """
        Should raise a validation error
        """
        ch = Channel(data_type=telem.DataType("CUSTOM"))
        seg = NumpySegment(
            channel=ch,
            start=telem.now(),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        with pytest.raises(errors.ValidationError):
            ScalarType().validate(seg)

    def test_invalid_array_dimensions(self):
        """
        Should raise a validation error
        """
        ch = Channel(data_type=telem.INT64)
        seg = NumpySegment(
            channel=ch,
            start=telem.now(),
            data=np.array([[1, 2, 3], [1, 2, 3]], dtype=np.int64),
        )
        with pytest.raises(errors.ValidationError):
            ScalarType().validate(seg)


class TestContiguityValidator:
    def test_valid_segment(self):
        """
        Should not raise a validation error
        """
        ch = Channel(
            key="1-1",
            data_type=telem.INT64,
            rate=25 * telem.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=telem.TimeStamp(100 * telem.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = Contiguity(
            {
                "1-1": telem.TimeStamp(100 * telem.SECOND),
            }
        )
        try:
            v.validate(seg)
        except Exception as e:
            pytest.fail(f"Unexpected exception: {e}")

    def test_multiple_segments_valid(self):
        """
        Should not raise a validation error
        """
        ch = Channel(
            key="1-1",
            data_type=telem.INT64,
            rate=1 * telem.HZ,
        )
        segs = [
            NumpySegment(
                channel=ch,
                start=telem.TimeStamp(100 * telem.SECOND),
                data=np.array([1, 2, 3], dtype=np.int64),
            ),
            NumpySegment(
                channel=ch,
                start=telem.TimeStamp(103 * telem.SECOND),
                data=np.array([1, 2, 3], dtype=np.int64),
            ),
        ]
        v = Contiguity(
            {
                "1-1": telem.TimeStamp(100 * telem.SECOND),
            }
        )
        for seg in segs:
            try:
                v.validate(seg)
            except Exception as e:
                pytest.fail(f"Unexpected exception: {e}")

    def test_overlapping_segment(self):
        """
        Should raise a contiguity error
        """
        ch = Channel(
            key="1-1",
            data_type=telem.INT64,
            rate=1 * telem.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=telem.TimeStamp(100 * telem.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = Contiguity(
            {
                "1-1": telem.TimeStamp(101 * telem.SECOND),
            }
        )
        with pytest.raises(errors.ContiguityError):
            v.validate(seg)

    def test_gapped_segment(self):
        """
        Should raise a contiguity error
        """
        ch = Channel(
            key="1-1",
            data_type=telem.INT64,
            rate=1 * telem.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=telem.TimeStamp(100 * telem.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = Contiguity(
            {
                "1-1": telem.TimeStamp(102 * telem.SECOND),
            }
        )
        with pytest.raises(errors.ContiguityError):
            v.validate(seg)

    def test_no_high_water_mark(self):
        """
        Should raise an unexpected error
        """
        ch = Channel(
            data_type=telem.INT64,
            rate=1 * telem.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=telem.TimeStamp(100 * telem.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = Contiguity({})
        with pytest.raises(errors.UnexpectedError):
            v.validate(seg)


class TestSplitter:
    def test_under_threshold_no_split(self):
        """
        Should return the original segment.
        """
        ch = Channel(
            data_type=telem.INT64,
            rate=1 * telem.HZ,
        )
        seg = SugaredBinarySegment(channel=ch, start=0, data=b"1234568")
        splitter = Splitter(threshold=telem.Size(16))
        split = splitter.split(seg)
        assert len(split) == 1

    def test_split_over_threshold(self):
        """
        Should split the segment when the size is over the threshold.
        """
        ch = Channel(
            data_type=telem.INT8,
            rate=1 * telem.HZ,
            density=telem.BIT8,
        )
        seg = SugaredBinarySegment(channel=ch, start=0, data=b"1234567812345678")
        splitter = Splitter(threshold=telem.Size(8))
        split = splitter.split(seg)
        assert len(split) == 2
        assert split[0].start == 0
        assert split[1].start == 8 * telem.SECOND
