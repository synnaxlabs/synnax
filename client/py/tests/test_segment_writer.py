import pytest
import numpy as np

import synnax.errors
from synnax import telem
from synnax.channel import Channel




class TestNumpy:
    def test_basic_write(self, channel: Channel, client: synnax.Client):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(10).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_invalid_data_type(self, channel: Channel, client: synnax.Client):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(10).astype(np.int64)
            with pytest.raises(synnax.errors.ValidationError):
                writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_invalid_data_shape(self, channel: Channel, client: synnax.Client):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(10, 10).astype(np.float64)
            with pytest.raises(synnax.errors.ValidationError):
                writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_non_contiguous_segments(self, channel: Channel, client: synnax.Client):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(10).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
            with pytest.raises(synnax.errors.ContiguityError):
                writer.write(to=channel.key, data=data, start=1)
        finally:
            writer.close()

    def test_multi_segment_write(self, channel: Channel, client: synnax.Client):
        writer = client.data.new_writer([channel.key])
        n_samples = 1000
        n_writes = 100
        try:
            for i in range(0, n_writes):
                data = np.random.rand(n_samples).astype(np.float64)
                writer.write(
                    to=channel.key, data=data, start=channel.rate.span(n_samples) * i
                )
        finally:
            writer.close()

    def test_segment_split(self, channel: Channel, client: synnax.Client):
        span = channel.rate.size_span(telem.Size(9e6), telem.BIT64)
        n_samples = channel.rate.sample_count(span)
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(n_samples).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()
