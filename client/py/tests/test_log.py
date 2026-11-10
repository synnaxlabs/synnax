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
from x.strings import random_name


class TestLog:
    def test_round_trip(self) -> None:
        """Should parse channel entries and dump them back unchanged."""
        lg = log.Log.model_validate(
            {"name": "Events", "channels": [{"channel": 1, "color": "#00ff00"}]}
        )
        entry = lg.channels[0]
        assert entry.precision is None
        assert entry.notation == "standard"
        assert entry.timestamp_config == log.TimestampConfig(
            format="preciseDate", time_zone="local"
        )
        assert log.Log.model_validate_json(lg.model_dump_json()) == lg

    def test_precision_bounds(self) -> None:
        """Should reject a precision above the float64 limit."""
        with pytest.raises(ValidationError):
            log.ChannelEntry.model_validate(
                {"channel": 1, "color": "#00ff00", "precision": 18}
            )

    def test_resource_id(self) -> None:
        """Should build the resource ID of a log."""
        key = uuid4()
        assert log.resource_id(key).type == log.TYPE_RESOURCE_ID.type
        assert log.resource_id(key).key == str(key)
        lg = log.Log(name="Events")
        assert hash(lg) == hash(lg.key)


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name="Log Project")


@pytest.mark.log
class TestClient:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a single log with default display configuration."""
        log = client.logs.create(sy.Log(name="Test Log", project=project.key))
        assert log.name == "Test Log"
        assert log.project == project.key
        assert log.channels == []
        assert log.timestamp_precision == 0
        assert not log.channel_names_hidden
        assert not log.timestamps_hidden

    def test_create_from_name(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create an empty log from a name."""
        log = client.logs.create(name="Named", project=project.key)
        assert log.name == "Named"
        assert log.channels == []

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple logs."""
        logs = client.logs.create(
            [
                sy.Log(name="Log 1", project=project.key),
                sy.Log(name="Log 2", project=project.key),
            ]
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
            timestamp_config=sy.log.TimestampConfig(format="ISO", time_zone="UTC"),
        )
        log = client.logs.create(
            sy.Log(
                name="Channels",
                channels=[entry],
                timestamp_precision=2,
                project=project.key,
            ),
        )
        retrieved = client.logs.retrieve(key=log.key)
        assert retrieved.channels == [entry]
        assert retrieved.timestamp_precision == 2

    def test_create_default_timestamp_config(
        self,
        client: sy.Synnax,
        project: sy.Project,
        virtual_channel: sy.Channel,
    ) -> None:
        """Should store the default timestamp config of a channel entry."""
        entry = sy.log.ChannelEntry(channel=virtual_channel.key, color=sy.Color())
        log = client.logs.create(
            sy.Log(name="Default", channels=[entry], project=project.key)
        )
        timestamp = client.logs.retrieve(key=log.key).channels[0].timestamp_config
        assert timestamp == sy.log.TimestampConfig(
            format="preciseDate", time_zone="local"
        )

    def test_retrieve_by_key(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve a log by key."""
        created = client.logs.create(sy.Log(name="Retrieve Me", project=project.key))
        retrieved = client.logs.retrieve(key=created.key)
        assert retrieved == created

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple logs by key."""
        created = client.logs.create(
            [
                sy.Log(name="Log A", project=project.key),
                sy.Log(name="Log B", project=project.key),
            ]
        )
        retrieved = client.logs.retrieve(keys=[log.key for log in created])
        assert {log.key for log in retrieved} == {log.key for log in created}

    def test_retrieve_by_project(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve only the logs the project owns."""
        a = client.logs.create(name="Child A", project=project.key)
        b = client.logs.create(name="Child B", project=project.key)
        other = client.projects.create(name="Other Project")
        client.logs.create(name="Other", project=other.key)
        retrieved = client.logs.retrieve(projects=[project.key])
        assert {r.key for r in retrieved} == {a.key, b.key}

    def test_retrieve_by_range_without_logs(
        self, client: sy.Synnax, project: sy.Project
    ) -> None:
        """Should return no logs when the range owns none."""
        client.logs.create(name="Project Owned", project=project.key)
        rng = client.ranges.create(
            name="Empty Range",
            time_range=sy.TimeStamp.now().span_range(sy.TimeSpan.SECOND),
        )
        assert client.logs.retrieve(ranges=[rng.key]) == []

    def test_retrieve_by_group(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should filter the logs by group."""
        made = client.logs.create(name="Ungrouped", project=project.key)
        g = client.groups.create("Empty", parent=sy.project.resource_id(project.key))
        assert client.logs.retrieve(groups=g.key) == []
        ungrouped = client.logs.retrieve(projects=project.key, grouped=False)
        assert [r.key for r in ungrouped] == [made.key]
        assert client.logs.retrieve(projects=project.key, grouped=True) == []

    def test_retrieve_where(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should keep the logs that match the where and the other filters."""
        a, b = client.logs.create(
            [
                sy.Log(name="A", project=project.key),
                sy.Log(name="B", project=project.key),
            ]
        )
        where = log.Where(not_=log.Where(key=sy.filter.Set[log.Key](eq=a.key)))
        res = client.logs.retrieve(where=where, keys=[a.key, b.key, uuid4()])
        assert [r.key for r in res] == [b.key]

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when a log does not exist."""
        with pytest.raises(NotFoundError):
            client.logs.retrieve(key=uuid4())

    def test_retrieve_key_list(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should return a list for a list of keys, even with one key."""
        a = client.logs.create(sy.Log(name="A", project=project.key))
        res = client.logs.retrieve(key=[a.key])
        assert [r.key for r in res] == [a.key]

    def test_retrieve_plain_key_beside_fields(
        self, client: sy.Synnax, project: sy.Project
    ) -> None:
        """Should return one log for a plain key beside other fields."""
        a = client.logs.create(sy.Log(name="A", project=project.key))
        assert client.logs.retrieve(key=a.key, not_={"key": uuid4()}).key == a.key
        with pytest.raises(NotFoundError, match=r"Found no log where .+"):
            client.logs.retrieve(key=a.key, not_={"key": a.key})

    def test_retrieve_or(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve the logs that match any filter in or_."""
        a = client.logs.create(sy.Log(name="A", project=project.key))
        b = client.logs.create(sy.Log(name="B", project=project.key))
        res = client.logs.retrieve(or_=[{"key": a.key}, {"key": [b.key]}])
        assert {r.key for r in res} == {a.key, b.key}

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a log."""
        log = client.logs.create(sy.Log(name="Before", project=project.key))
        client.logs.rename(log.key, "After")
        assert client.logs.retrieve(key=log.key).name == "After"

    def test_add_channel(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should add a channel once with the default display configuration."""
        log = client.logs.create(name="Add", project=project.key)
        client.logs.add_channel(log.key, virtual_channel.key)
        client.logs.add_channel(log.key, virtual_channel.key)
        assert client.logs.retrieve(key=log.key).channels == [
            sy.log.ChannelEntry(channel=virtual_channel.key, color=sy.Color(a=0))
        ]

    def test_remove_channel(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should remove a channel from a log."""
        log = client.logs.create(name="Remove", project=project.key)
        client.logs.add_channel(log.key, virtual_channel.key)
        client.logs.remove_channel(log.key, virtual_channel.key)
        assert client.logs.retrieve(key=log.key).channels == []

    def test_set_channel_fields(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should set the color, alias, notation, and precision of a channel."""
        log = client.logs.create(name="Edit", project=project.key)
        ch = virtual_channel.key
        client.logs.add_channel(log.key, ch)
        client.logs.set_channel_color(log.key, ch, "#FF0000")
        client.logs.set_channel_alias(log.key, ch, "pressure")
        client.logs.set_channel_notation(log.key, ch, "scientific")
        client.logs.set_channel_precision(log.key, ch, 3)
        entry = client.logs.retrieve(key=log.key).channels[0]
        assert entry.color == "#FF0000"
        assert entry.alias == "pressure"
        assert entry.notation == "scientific"
        assert entry.precision == 3

    def test_clear_channel_precision(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should clear the precision of a channel when given None."""
        log = client.logs.create(name="Clear", project=project.key)
        client.logs.add_channel(log.key, virtual_channel.key)
        client.logs.set_channel_precision(log.key, virtual_channel.key, 2)
        client.logs.set_channel_precision(log.key, virtual_channel.key, None)
        assert client.logs.retrieve(key=log.key).channels[0].precision is None

    def test_set_channel_precision_out_of_bounds(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should reject a precision above 17."""
        log = client.logs.create(name="Bounds", project=project.key)
        client.logs.add_channel(log.key, virtual_channel.key)
        with pytest.raises(ValidationError):
            client.logs.set_channel_precision(log.key, virtual_channel.key, 18)

    def test_set_channel_entry(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should add a channel entry, then replace it in place."""
        log = client.logs.create(name="Entry", project=project.key)
        entry = sy.log.ChannelEntry(
            channel=virtual_channel.key, color=sy.Color("#00FF00"), alias="first"
        )
        client.logs.set_channel_entry(log.key, entry)
        assert client.logs.retrieve(key=log.key).channels == [entry]
        replaced = entry.model_copy(update={"alias": "second", "precision": 4})
        client.logs.set_channel_entry(log.key, replaced)
        assert client.logs.retrieve(key=log.key).channels == [replaced]

    def test_set_channel_timestamp_config(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should set the timestamp format and time zone of a channel."""
        log = client.logs.create(name="Timestamp", project=project.key)
        ch = virtual_channel.key
        client.logs.add_channel(log.key, ch)
        client.logs.set_channel_timestamp_format(log.key, ch, "ISO")
        client.logs.set_channel_timestamp_time_zone(log.key, ch, "UTC")
        entry = client.logs.retrieve(key=log.key).channels[0]
        assert entry.timestamp_config == sy.log.TimestampConfig(
            format="ISO", time_zone="UTC"
        )

    def test_swap_channel(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should replace a channel and keep its display configuration."""
        other = client.channels.create(
            name=f"test_{random_name()}_swap",
            virtual=True,
            data_type=sy.DataType.FLOAT64,
        )
        log = client.logs.create(name="Swap", project=project.key)
        client.logs.add_channel(log.key, virtual_channel.key)
        client.logs.set_channel_alias(log.key, virtual_channel.key, "kept")
        client.logs.swap_channel(log.key, virtual_channel.key, other.key)
        [entry] = client.logs.retrieve(key=log.key).channels
        assert entry.channel == other.key
        assert entry.alias == "kept"

    def test_set_display_settings(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should set the timestamp precision, hidden flags, and time zone."""
        log = client.logs.create(name="Display", project=project.key)
        client.logs.set_timestamp_precision(log.key, 3)
        client.logs.set_channel_names_hidden(log.key, True)
        client.logs.set_timestamps_hidden(log.key, True)
        client.logs.set_time_zone(log.key, "UTC")
        res = client.logs.retrieve(key=log.key)
        assert res.timestamp_precision == 3
        assert res.channel_names_hidden
        assert res.timestamps_hidden
        assert res.time_zone == "UTC"

    def test_set_timestamp_precision_out_of_bounds(
        self, client: sy.Synnax, project: sy.Project
    ) -> None:
        """Should reject a timestamp precision above 3."""
        log = client.logs.create(name="Bounds", project=project.key)
        with pytest.raises(ValidationError):
            client.logs.set_timestamp_precision(log.key, 4)

    def test_set_missing_channel(
        self, client: sy.Synnax, project: sy.Project, virtual_channel: sy.Channel
    ) -> None:
        """Should leave a log unchanged when it does not show the channel."""
        log = client.logs.create(name="Missing", project=project.key)
        client.logs.set_channel_alias(log.key, virtual_channel.key, "ignored")
        assert client.logs.retrieve(key=log.key).channels == []

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete a log."""
        log = client.logs.create(sy.Log(name="To Delete", project=project.key))
        client.logs.delete(log.key)
        with pytest.raises(NotFoundError):
            client.logs.retrieve(key=log.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple logs."""
        logs = client.logs.create(
            [
                sy.Log(name="Delete 1", project=project.key),
                sy.Log(name="Delete 2", project=project.key),
            ]
        )
        client.logs.delete([log.key for log in logs])
        assert client.logs.retrieve(keys=[log.key for log in logs]) == []

    def test_delete_missing(self, client: sy.Synnax) -> None:
        """Should not raise when deleting a log that does not exist."""
        client.logs.delete(uuid4())
