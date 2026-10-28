#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest
from pydantic import ValidationError

import synnax as sy
from synnax import log
from synnax.exceptions import NotFoundError


class TestLog:
    def test_round_trip(self) -> None:
        """Should parse channel entries and dump them back unchanged."""
        lg = log.Log.model_validate(
            {"name": "Events", "channels": [{"channel": 1, "color": "#00ff00"}]}
        )
        entry = lg.channels[0]
        assert entry.precision == -1
        assert entry.notation == "standard"
        assert entry.timestamp == log.TimestampConfig(format="preciseDate", tz="local")
        assert log.Log.model_validate_json(lg.model_dump_json()) == lg

    def test_precision_bounds(self) -> None:
        """Should reject a precision above the float64 limit."""
        with pytest.raises(ValidationError):
            log.ChannelEntry.model_validate(
                {"channel": 1, "color": "#00ff00", "precision": 18}
            )

    def test_ontology_id(self) -> None:
        """Should build the ontology ID of a log."""
        key = uuid4()
        assert log.ontology_id(key).type == log.ONTOLOGY_TYPE.type
        assert log.ontology_id(key).key == str(key)
        lg = log.Log(name="Events")
        assert hash(lg) == hash(lg.key)


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name="Log Project")


@pytest.mark.log
class TestClient:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a single log with default display configuration."""
        log = client.logs.create(project.key, sy.Log(name="Test Log"))
        assert log.name == "Test Log"
        assert log.channels == []
        assert log.timestamp_precision == 0
        assert not log.channel_names_hidden
        assert not log.receipt_timestamp_hidden

    def test_create_from_name(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create an empty log from a name."""
        log = client.logs.create(project.key, name="Named")
        assert log.name == "Named"
        assert log.channels == []

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple logs."""
        logs = client.logs.create(
            project.key, [sy.Log(name="Log 1"), sy.Log(name="Log 2")]
        )
        assert [log.name for log in logs] == ["Log 1", "Log 2"]

    def test_create_with_channels(
        self,
        client: sy.Synnax,
        project: sy.Project,
        virtual_channel: sy.Channel,
    ) -> None:
        """Should keep the channel entries of a created log."""
        entry = sy.log.ChannelEntry(
            channel=virtual_channel.key,
            color=sy.Color("#FF0000"),
            alias="pressure",
            timestamp=sy.log.TimestampConfig(format="ISO", tz="UTC"),
        )
        log = client.logs.create(
            project.key,
            sy.Log(name="Channels", channels=[entry], timestamp_precision=2),
        )
        retrieved = client.logs.retrieve(key=log.key)
        assert retrieved.channels == [entry]
        assert retrieved.timestamp_precision == 2

    def test_create_default_timestamp(
        self,
        client: sy.Synnax,
        project: sy.Project,
        virtual_channel: sy.Channel,
    ) -> None:
        """Should store the default timestamp configuration of a channel entry."""
        entry = sy.log.ChannelEntry(channel=virtual_channel.key, color=sy.Color())
        log = client.logs.create(project.key, sy.Log(name="Default", channels=[entry]))
        timestamp = client.logs.retrieve(key=log.key).channels[0].timestamp
        assert timestamp == sy.log.TimestampConfig(format="preciseDate", tz="local")

    def test_retrieve_by_key(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve a log by key."""
        created = client.logs.create(project.key, sy.Log(name="Retrieve Me"))
        retrieved = client.logs.retrieve(key=created.key)
        assert retrieved == created

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple logs by key."""
        created = client.logs.create(
            project.key, [sy.Log(name="Log A"), sy.Log(name="Log B")]
        )
        retrieved = client.logs.retrieve(keys=[log.key for log in created])
        assert {log.key for log in retrieved} == {log.key for log in created}

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when a log does not exist."""
        with pytest.raises(NotFoundError):
            client.logs.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a log."""
        log = client.logs.create(project.key, sy.Log(name="Before"))
        client.logs.rename(log.key, "After")
        assert client.logs.retrieve(key=log.key).name == "After"

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete a log."""
        log = client.logs.create(project.key, sy.Log(name="To Delete"))
        client.logs.delete(log.key)
        with pytest.raises(NotFoundError):
            client.logs.retrieve(key=log.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple logs."""
        logs = client.logs.create(
            project.key, [sy.Log(name="Delete 1"), sy.Log(name="Delete 2")]
        )
        client.logs.delete([log.key for log in logs])
        with pytest.raises(NotFoundError):
            client.logs.retrieve(keys=[log.key for log in logs])

    def test_delete_missing(self, client: sy.Synnax) -> None:
        """Should not raise when deleting a log that does not exist."""
        client.logs.delete(uuid4())
