import pytest

import synnax
from synnax import telem
from synnax.channel import Channel


class TestClient:
    @pytest.fixture(scope="class")
    def two_channels(self, client: synnax.Client) -> list[Channel]:
        ch = Channel(
            name="test",
            node_id=1,
            rate=25 * telem.HZ,
            data_type=telem.FLOAT64,
            density=telem.BIT64,
        )
        return client.channel.create_n(ch, 2)

    def test_create(self, two_channels: list[Channel]):
        assert len(two_channels) == 2
        for channel in two_channels:
            assert channel.name == "test"
            assert channel.key != ""

    def test_retrieve_by_key(
        self, two_channels: list[Channel], client: synnax.Client
    ) -> None:
        res_channels = client.channel.retrieve(
            [channel.key for channel in two_channels]
        )
        assert len(res_channels) == 2
        for i, channel in enumerate(res_channels):
            assert two_channels[i].key == channel.key
            assert isinstance(two_channels[i].density, telem.Density)

    def test_retrieve_by_node_id(
        self, two_channels: list[Channel], client: synnax.Client
    ) -> None:
        res_channels = client.channel.retrieve_by_node_id(1)
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.node_id == 1

    def test_create_missing_field(self, client: synnax.Client):
        with pytest.raises(synnax.errors.ValidationError):
            client.channel.create(
                name="my_channel",
                node_id=2,
            )