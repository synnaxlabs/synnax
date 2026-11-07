#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from x.strings import random_name


@pytest.fixture
def sift_device(client: sy.Synnax) -> sy.Device:
    """A Sift device on the rack of the Core, with an unreachable URI."""
    api_key = client.secrets.create(name="Sift API key", value="key").key
    rack = client.racks.retrieve(integrations=sy.sift.INTEGRATION)[0]
    return client.devices.create(
        sy.sift.Device(
            uri="localhost:1", api_key=api_key, rack=rack.key, name=random_name()
        )
    )


@pytest.mark.sift
class TestSiftStreamTask:
    def test_config_defaults(self) -> None:
        """Should fill the config from the arguments and default the rest."""
        tsk = sy.sift.StreamTask(device="dev", asset="Stand 1", channels=[1, 2])
        assert tsk.config.device == "dev"
        assert tsk.config.asset == "Stand 1"
        assert tsk.config.channels == [1, 2]
        assert tsk.config.labels == []
        assert tsk.config.auto_start is False

    def test_parse_internal(self) -> None:
        """Should parse the config of a stored task."""
        internal = sy.Task(
            name="stream",
            type=sy.sift.StreamTask.TYPE,
            config={"device": "dev", "asset": "Stand 1", "channels": [3]},
        )
        assert sy.sift.StreamTask(internal).config.channels == [3]


@pytest.mark.sift
class TestSiftDevice:
    def test_properties(self) -> None:
        """Should hold the URI and the key of the API key secret."""
        key = sy.secret.Key("6f1c2a4e-8b3d-4c5f-9a7e-0d2b4c6e8f1a")
        dev = sy.sift.Device(uri="localhost:50051", api_key=key, rack=1)
        assert dev.make == sy.sift.MAKE
        assert dev.properties == {"uri": "localhost:50051", "api_key": str(key)}
        props = sy.sift.Properties.model_validate(dev.properties)
        assert props.api_key == key


@pytest.mark.sift
class TestSiftUpload:
    def test_no_range_or_time_range(
        self, client: sy.Synnax, sift_device: sy.Device
    ) -> None:
        """Should raise when the upload names no data."""
        with pytest.raises(RuntimeError, match="a range or a time range is required"):
            sy.sift.upload(client.tasks, sift_device, asset="Stand 1", channels=[1])

    def test_future_time_range(self, client: sy.Synnax, sift_device: sy.Device) -> None:
        """Should send the time range to the Core, which rejects a future one."""
        start = sy.TimeStamp.now() + sy.TimeSpan.HOUR
        with pytest.raises(RuntimeError, match="no time span to upload"):
            sy.sift.upload(
                client.tasks,
                sift_device,
                asset="Stand 1",
                channels=[1],
                time_range=sy.TimeRange(start, start + sy.TimeSpan.SECOND),
            )
