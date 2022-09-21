import numpy as np
import pytest
from freighter import EncoderDecoder, MsgpackEncoder

import synnax
from synnax.channel.payload import ChannelPayload
from synnax.segment.payload import SegmentPayload
from synnax.segment.splitter import Splitter
from synnax.segment.sugared import NumpySegment, SugaredBinarySegment
from synnax.segment.validate import ContiguityValidator, ScalarTypeValidator


class TestSegmentPayload:
    @pytest.mark.parametrize("ecd", [MsgpackEncoder()])
    def test_encode_decode(self, ecd: EncoderDecoder):
        segment = SegmentPayload(
            channel_key="1-1",
            start=synnax.TimeStamp(1),
            data=b"12345",
        )
        encoded = ecd.encode(segment)
        decoded = ecd.decode(encoded, SegmentPayload)
        assert segment.dict() == decoded.dict()


class TestScalarTypeValidator:
    def test_valid_segment(self):
        """
        Should not raise a validation error
        """
        ch = ChannelPayload(data_type=synnax.INT64, rate=0)
        seg = NumpySegment(
            channel=ch,
            start=synnax.now(),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        try:
            ScalarTypeValidator().validate(seg)
        except Exception as e:
            pytest.fail(f"Unexpected exception: {e}")

    def test_invalid_segment(self):
        """
        Should raise a validation error
        """
        ch = ChannelPayload(data_type=synnax.INT64, rate=0)
        seg = NumpySegment(
            channel=ch,
            start=synnax.now(),
            data=np.array([1, 2, 3], dtype=np.int32),
        )
        with pytest.raises(synnax.ValidationError):
            ScalarTypeValidator().validate(seg)

    def test_unrecognized_data_type(self):
        """
        Should raise a validation error
        """
        ch = ChannelPayload(data_type=synnax.DataType("CUSTOM"), rate=0)
        seg = NumpySegment(
            channel=ch,
            start=synnax.now(),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        with pytest.raises(synnax.ValidationError):
            ScalarTypeValidator().validate(seg)

    def test_invalid_array_dimensions(self):
        """
        Should raise a validation error
        """
        ch = ChannelPayload(data_type=synnax.INT64, rate=0)
        seg = NumpySegment(
            channel=ch,
            start=synnax.now(),
            data=np.array([[1, 2, 3], [1, 2, 3]], dtype=np.int64),
        )
        with pytest.raises(synnax.ValidationError):
            ScalarTypeValidator().validate(seg)


class TestContiguityValidator:
    def test_valid_segment(self):
        """
        Should not raise a validation error
        """
        ch = ChannelPayload(
            key="1-1",
            data_type=synnax.INT64,
            rate=25 * synnax.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=synnax.TimeStamp(100 * synnax.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = ContiguityValidator(
            {
                "1-1": synnax.TimeStamp(100 * synnax.SECOND),
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
        ch = ChannelPayload(
            key="1-1",
            data_type=synnax.INT64,
            rate=1 * synnax.HZ,
        )
        segs = [
            NumpySegment(
                channel=ch,
                start=synnax.TimeStamp(100 * synnax.SECOND),
                data=np.array([1, 2, 3], dtype=np.int64),
            ),
            NumpySegment(
                channel=ch,
                start=synnax.TimeStamp(103 * synnax.SECOND),
                data=np.array([1, 2, 3], dtype=np.int64),
            ),
        ]
        v = ContiguityValidator(
            {
                "1-1": synnax.TimeStamp(100 * synnax.SECOND),
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
        ch = ChannelPayload(
            key="1-1",
            data_type=synnax.INT64,
            rate=1 * synnax.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=synnax.TimeStamp(100 * synnax.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = ContiguityValidator(
            {
                "1-1": synnax.TimeStamp(101 * synnax.SECOND),
            }
        )
        with pytest.raises(synnax.ContiguityError):
            v.validate(seg)

    def test_gapped_segment(self):
        """
        Should raise a contiguity error
        """
        ch = ChannelPayload(
            key="1-1",
            data_type=synnax.INT64,
            rate=1 * synnax.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=synnax.TimeStamp(100 * synnax.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = ContiguityValidator(
            {
                "1-1": synnax.TimeStamp(102 * synnax.SECOND),
            }
        )
        with pytest.raises(synnax.ContiguityError):
            v.validate(seg)

    def test_no_high_water_mark(self):
        """
        Should raise an unexpected error
        """
        ch = ChannelPayload(
            data_type=synnax.INT64,
            rate=1 * synnax.HZ,
        )
        seg = NumpySegment(
            channel=ch,
            start=synnax.TimeStamp(100 * synnax.SECOND),
            data=np.array([1, 2, 3], dtype=np.int64),
        )
        v = ContiguityValidator({})
        with pytest.raises(synnax.UnexpectedError):
            v.validate(seg)


class TestSplitter:
    def test_under_threshold_no_split(self):
        """
        Should return the original segment.
        """
        ch = ChannelPayload(
            data_type=synnax.INT64,
            rate=1 * synnax.HZ,
        )
        seg = SugaredBinarySegment(channel=ch, start=0, data=b"1234568")
        splitter = Splitter(threshold=synnax.Size(16))
        split = splitter.split(seg)
        assert len(split) == 1

    def test_split_over_threshold(self):
        """
        Should split the segment when the size is over the threshold.
        """
        ch = ChannelPayload(
            data_type=synnax.INT8,
            rate=1 * synnax.HZ,
            density=synnax.BIT8,
        )
        seg = SugaredBinarySegment(channel=ch, start=0, data=b"1234567812345678")
        splitter = Splitter(threshold=synnax.Size(8))
        split = splitter.split(seg)
        assert len(split) == 2
        assert split[0].start == 0
        assert split[1].start == 8 * synnax.SECOND
