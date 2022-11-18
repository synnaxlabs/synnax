import numpy as np
import pytest

import synnax


class TestNumpy:
    def test_basic_write(self, channel: synnax.Channel, client: synnax.Synnax):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(10).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_invalid_data_type(self, channel: synnax.Channel, client: synnax.Synnax):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(10).astype(np.int64)
            with pytest.raises(synnax.ValidationError):
                writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_invalid_data_shape(self, channel: synnax.Channel, client: synnax.Synnax):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(10, 10).astype(np.float64)
            with pytest.raises(synnax.ValidationError):
                writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()

    def test_multi_segment_write(self, channel: synnax.Channel, client: synnax.Synnax):
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

    def test_segment_split(self, channel: synnax.Channel, client: synnax.Synnax):
        span = channel.rate.size_span(synnax.Size(9e6), synnax.BIT64)
        n_samples = channel.rate.sample_count(span)
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(n_samples).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
        finally:
            writer.close()
