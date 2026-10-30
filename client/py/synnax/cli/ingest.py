#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path
from typing import Any

import click
import numpy as np

from synnax import Synnax
from synnax.channel import Channel
from synnax.cli import channel, connect, default, flow
from synnax.cli.flow import Context
from synnax.cli.io import prompt_file
from synnax.cli.telem import select_data_type
from synnax.config import default_clusters
from synnax.ingest import row
from synnax.io import FACTORY, ChannelMeta, Factory, ReaderType, RowFileReader
from synnax.telem import DataType, Rate, TimeRange, TimeStamp

# A shorthand grouping to represent all values in a set.
_GROUP_ALL = "__all__"


@click.command()
@click.argument("path_", type=click.Path(exists=True), required=False, default=None)
def ingest(path_: str | None) -> None:
    return _pure_ingest(path_, ctx=default.context())


def _pure_ingest(
    path_: Path | str | None,
    ctx: Context,
    client: Synnax | None = None,
) -> None:
    runner: flow.Flow[Any] = flow.Flow(ctx)
    runner.add("initialize_reader", _initialize_reader)
    runner.add("connect_client", _connect_client)
    runner.add("ingest_all", _ingest_all)
    runner.add("channels_to_ingest", _channels_to_ingest)
    runner.add("skip_invalid_channels", _skip_invalid_channels)
    runner.add("validate_channels_exist", _validate_channels_exist)
    runner.add("validate_data_types", _validate_data_types)
    runner.add("validate_start_time", _validate_start_time)
    runner.add("create_channels", _create_channels)
    runner.add("prompt_name", _prompt_name)
    runner.add("ingest", _run_ingestion)
    path = None if path_ is None else Path(path_)
    runner.run(_Flow(FACTORY, path, client), root="initialize_reader")


class _Flow:
    name: str | None = None
    path: Path | None = None
    factory: Factory
    reader: RowFileReader | None
    client: Synnax | None
    filtered_channels: list[ChannelMeta] | None
    not_found: list[ChannelMeta] | None
    db_channels: list[Channel] | None
    start: TimeStamp | None = None

    def __init__(self, factory: Factory, path: Path | None, client: Synnax | None):
        self.path = path
        self.factory = factory
        self.reader = None
        self.client = client
        self.filtered_channels = None
        self.not_found = None
        self.db_channels = None


def _run_ingestion(ctx: Context, cli: _Flow) -> None:
    assert cli.reader is not None
    assert cli.db_channels is not None
    assert cli.client is not None
    assert cli.start is not None
    _initialize_reader(ctx, cli)
    if cli.reader.type() == ReaderType.Row:
        engine = row.Engine(
            client=cli.client,
            reader=cli.reader,
            channels=cli.db_channels,
            start=cli.start,
        )
    else:
        raise NotImplementedError("Only row ingestion is supported at this time.")
    ctx.console.info("Starting ingestion process...")
    engine.run()
    assert cli.name is not None
    cli.client.ranges.create(name=cli.name, time_range=TimeRange(cli.start, engine.end))


def _initialize_reader(
    ctx: Context,
    cli: _Flow,
) -> str | None:
    ctx.console.info("Welcome to the Synnax ingestion CLI! Let's get started.")
    if cli.path is None:
        ctx.console.info("Please select a file to ingest.")
        cli.path = prompt_file(ctx)
    if cli.path is None:
        return None
    cli.reader = cli.factory.new_reader(cli.path)
    return "connect_client"


def _connect_client(ctx: Context, cli: _Flow) -> str | None:
    if cli.client is None:
        cli.client = connect.from_clusters(ctx, default_clusters())
    return "ingest_all" if cli.client else None


def _ingest_all(ctx: Context, cli: _Flow) -> str | None:
    assert cli.reader is not None
    if ctx.console.ask("Would you like to ingest all channels?", default=True):
        cli.filtered_channels = cli.reader.channels()
        return "skip_invalid_channels"
    return "channels_to_ingest"


def _channels_to_ingest(ctx: Context, cli: _Flow) -> str | None:
    assert cli.reader is not None
    ctx.console.info("Which channels would you like to ingest?")
    channels = cli.reader.channels()
    grouped = channel.prompt_group_names(ctx, [ch.name for ch in channels])
    if grouped is None or len(grouped) == 0:
        return None
    all_names = [name for names in grouped.values() for name in names]
    cli.filtered_channels = [ch for ch in channels if ch.name in all_names]
    return "skip_invalid_channels"


def _skip_invalid_channels(ctx: Context, cli: _Flow) -> str | None:
    assert cli.reader is not None
    data_types = {
        key: dt
        for key, dt in _read_data_types(ctx, cli).items()
        if dt == DataType.UNKNOWN
    }
    if len(data_types) > 0:
        ctx.console.info("The following channels have non-numeric data types")
        channel.name_table(ctx, list(data_types))
        if not ctx.console.ask("Skip these channels?", default=True):
            return None
        assert cli.filtered_channels is not None
        cli.filtered_channels = [
            ch for ch in cli.filtered_channels if ch.name not in data_types
        ]
    return "validate_channels_exist"


def _cannot_cast_error(ctx: Context, actual: Any, ch: Channel) -> None:
    ctx.console.error(f"""Unable to cast column data type {actual}
for {ch.name} to channel data type {ch.data_type.np}""")


def _validate_data_types(ctx: Context, cli: _Flow) -> str | None:
    """Does an optimistic check on the first sample of each channel.  This isn't error
    prone, but checking every sample in a large file would be too slow.
    """
    assert cli.db_channels is not None
    d_types = _read_data_types(ctx, cli)

    for ch in cli.db_channels:
        samples_type = d_types[ch.name].np
        ch_type = ch.data_type.np
        if not np.can_cast(samples_type, ch_type):
            _cannot_cast_error(ctx, samples_type, ch)
            return None
        if samples_type != ch_type:
            ctx.console.warn(
                f"""Channel {ch.name} has data type {ch_type} but the file data type is
                {samples_type}. Synnax can cast between them safely, but you have been
                warned."""
            )

    return "validate_start_time"


def _read_data_types(ctx: Context, cli: _Flow) -> dict[str, DataType]:
    assert cli.reader is not None
    assert cli.filtered_channels is not None

    cli.reader.set_chunk_size(1)
    cli.reader.seek_first()

    first = cli.reader.read()

    data_types = {}
    for ch in cli.filtered_channels:
        samples = first[ch.name]
        if len(samples) == 0:
            ctx.console.warn(f"Channel {ch.name} has no samples")
            continue
        try:
            data_types[ch.name] = DataType(samples.to_numpy().dtype)
        except TypeError:
            data_types[ch.name] = DataType.UNKNOWN
    return data_types


def _validate_channels_exist(ctx: Context, cli: _Flow) -> str | None:
    """Validates that all channels in the file exist in the database. If not, prompts
    the user to create them.
    """
    assert cli.filtered_channels is not None
    assert cli.client is not None

    ctx.console.info("Validating that channels exist...")
    cli.not_found = []
    cli.db_channels = []
    for meta in cli.filtered_channels:
        ch = channel.maybe_select(
            ctx,
            cli.client.channels.retrieve([meta.name]),
            param=meta.name,
        )
        if ch is None:
            cli.not_found.append(meta)
        else:
            cli.db_channels.append(ch)

    if len(cli.not_found) > 0:
        ctx.console.info("The following channels were not found in the database:")
        channel.name_table(ctx, [ch.name for ch in cli.not_found])
        if not ctx.console.ask("Would you like to create them?", default=True):
            return None
        return "create_channels"

    return "validate_data_types"


def _validate_start_time(ctx: Context, cli: _Flow) -> str | None:
    """Reads the starting timestamp of the file and prompts the user to confirm that it
    is correct. This is mainly a sanity check to make sure the timestamps are properly
    formatted."""
    assert cli.db_channels is not None
    assert cli.reader is not None
    _idx = [ch for ch in cli.db_channels if ch.is_index]
    if len(_idx) == 0:
        # If there is no index, it means all channels are rate based or we've already
        # written the index data. In either case, we need to prompt the user to enter
        # the start timestamp.
        _start = ctx.console.ask(
            """Please enter the start timestamp of the file as a
            nanosecond UTC integer. If you'd like a converter,
            use https://www.epochconverter.com/""",
            type_=int,
            default=int(TimeStamp.now()),
        )
        assert _start is not None
        cli.start = TimeStamp(_start)
    else:
        idx = _idx[0]
        cli.reader.set_chunk_size(1)
        cli.reader.seek_first()
        first = cli.reader.read()
        cli.start = TimeStamp(first[idx.name].to_numpy()[0])

    ctx.console.info(f"Identified start timestamp for file as {cli.start}.")
    if not ctx.console.ask("Is this correct?", default=True):
        return None
    return "prompt_name"


def _create_indexes(
    ctx: Context,
    cli: _Flow,
    options: list[ChannelMeta],
) -> list[ChannelMeta]:
    assert cli.client is not None
    assert cli.db_channels is not None
    grouped = channel.prompt_group_names(ctx, [ch.name for ch in options])
    if grouped is None:
        return options
    names = [name for v in grouped.values() for name in v]
    for name in names:
        ch = cli.client.channels.create(
            name=name, is_index=True, data_type=DataType.TIMESTAMP
        )
        cli.db_channels.append(ch)
    return [ch for ch in options if ch.name not in names]


_DATA_TYPE_OPTIONS = [
    "Guess data types from file",
    "Assign the same data type to all channels (excluding indexes)",
    "Group channels by data type",
]


def _assign_data_type(
    ctx: Context,
    cli: _Flow,
) -> dict[DataType, list[ChannelMeta]] | None:
    assert cli.not_found is not None

    grouped: dict[str, list[ChannelMeta]] = {_GROUP_ALL: cli.not_found}
    assigned: dict[DataType, list[ChannelMeta]] = {}
    ctx.console.info("Please select an option for assigning data types:")
    opt, _ = ctx.console.select(
        rows=_DATA_TYPE_OPTIONS,
        default=_DATA_TYPE_OPTIONS[0],
    )
    if opt == _DATA_TYPE_OPTIONS[0]:
        data_types = _read_data_types(ctx, cli)
        for ch in cli.not_found:
            dt = data_types[ch.name]
            if dt not in assigned:
                assigned[dt] = [ch]
            else:
                assigned[dt].append(ch)
        return assigned
    if opt == _DATA_TYPE_OPTIONS[2]:
        groups = channel.prompt_group_names(ctx, [ch.name for ch in cli.not_found])
        if groups is None or len(groups) == 0:
            return None
        grouped = {
            k: [ch for ch in cli.not_found if ch.name in v] for k, v in groups.items()
        }
    for key, group in grouped.items():
        if key != _GROUP_ALL:
            ctx.console.info(f"Assigning data type to {key}")
        selected_dt = select_data_type(ctx)
        if selected_dt is None:
            return None
        assigned[selected_dt] = group
    return assigned


def _assign_index_or_rate(
    ctx: Context,
    cli: _Flow,
) -> dict[Rate | int, list[ChannelMeta]] | None:
    assert cli.client is not None
    assert cli.not_found is not None
    client = cli.client

    grouped: dict[str, list[ChannelMeta]] = {_GROUP_ALL: cli.not_found}
    if not ctx.console.ask(
        "Do all non-indexed channels have the same data rate or index?",
        type_=bool,
        default=True,
    ):
        if not ctx.console.ask(
            "Can you group channels by data rate or index?", default=True
        ):
            grouped = {v.name: [v] for v in cli.not_found}
        groups = channel.prompt_group_names(ctx, [ch.name for ch in cli.not_found])
        if groups is None or len(groups) == 0:
            return None
        grouped = {
            k: [ch for ch in cli.not_found if ch.name in v] for k, v in groups.items()
        }

    def assign_to_group(key: str, group: list[ChannelMeta]) -> Rate | int | None:
        if key != _GROUP_ALL:
            ctx.console.info(f"Assigning data rate or index to {key}")
        _choice = ctx.console.ask("Enter the name of an index or a data rate")
        assert _choice is not None
        if _choice.replace(".", "").isdigit():
            return Rate(float(_choice))
        # If the user entered a string, we have an index channel, and we
        # need to make sure that the string is a valid index.
        res = client.channels.retrieve([_choice])
        idx = channel.maybe_select(ctx, res, param=_choice)
        if not idx:
            ctx.console.warn(f"Index channel with key {_choice} not found")
            if ctx.console.ask("Try again?", default=True):
                return assign_to_group(key, group)
            return None
        return idx.key

    assigned: dict[Rate | int, list[ChannelMeta]] = {}
    for key, group in grouped.items():
        idx = assign_to_group(key, group)
        if idx is None:
            return None
        assigned[idx] = group

    return assigned


def _create_channels(ctx: Context, cli: _Flow) -> str | None:
    assert cli.not_found is not None
    assert cli.client is not None
    assert cli.db_channels is not None

    if ctx.console.ask("Are any channels indexed (e.g. timestamps)?", default=True):
        cli.not_found = _create_indexes(ctx, cli, cli.not_found)

    idx_grouped, dt_grouped = (
        _assign_index_or_rate(ctx, cli),
        _assign_data_type(ctx, cli),
    )
    if idx_grouped is None or dt_grouped is None:
        return None

    to_create: list[Channel] = []
    for rate_or_index, channels in idx_grouped.items():
        index = 0 if isinstance(rate_or_index, Rate) else rate_or_index
        to_create.extend(
            Channel(
                name=ch.name,
                is_index=False,
                index=index,
                data_type=next(dt for dt, chs in dt_grouped.items() if ch in chs),
            )
            for ch in channels
        )

    cli.db_channels.extend(cli.client.channels.create(to_create))

    return "validate_data_types"


def _prompt_name(ctx: Context, cli: _Flow) -> str | None:
    assert cli.db_channels is not None
    assert cli.not_found is not None
    assert cli.client is not None
    assert cli.path is not None
    path: Path = cli.path
    ctx.console.info("Please enter a name for the data set")
    cli.name = ctx.console.ask("Name", default=path.name)
    return "ingest"
