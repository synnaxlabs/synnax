#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
from pathlib import Path

import click

from synnax import Synnax
from synnax.cli.connect import connect_client
from synnax.cli.channel import (
    prompt_group_channel_names,
    channel_name_table,
    maybe_select_channel,
)
from synnax.ingest.row import RowIngestionEngine
from synnax.io import ChannelMeta, IOFactory, ReaderType, RowReader
from synnax.telem import TIMESTAMP, Rate, TimeStamp
from synnax.channel import Channel
from synnax.cli.console.rich import RichConsole
from synnax.cli.flow import Context, Flow
from synnax.cli.io import prompt_file
from synnax.cli.telem import prompt_data_type_select

# A shorthand grouping to represent all values in a set.
GROUP_ALL = "__all__"


@click.command()
@click.argument("_path", type=click.Path(exists=True), required=False, default=None)
def ingest(_path: str | None):
    flow = Flow(Context(console=RichConsole()))
    flow.add("initialize_reader", initialize_reader)
    flow.add("connect_client", _connect_client)
    flow.add("ingest_all", ingest_all)
    flow.add("channels_to_ingest", channels_to_ingest)
    flow.add("validate_channels_exist", validate_channels_exist)
    flow.add("validate_data_types", validate_data_types)
    flow.add("validate_start_time", validate_start_time)
    flow.add("create_channels", create_channels)
    flow.add("ingest", run_ingestion)
    path = None if _path is None else Path(_path)
    flow.run(IngestionCLI(IOFactory(), path), "initialize_reader")


class IngestionCLI:
    path: Path | None = None
    factory: IOFactory
    reader: RowReader | None
    client: Synnax | None
    filtered_channels: list[ChannelMeta] | None
    not_found: list[ChannelMeta] | None
    db_channels: list[Channel] | None
    start: TimeStamp | None = None

    def __init__(self, factory: IOFactory, path: Path | None):
        self.path = path
        self.factory = factory
        self.reader = None
        self.client = None
        self.filtered_channels = None
        self.not_found = None
        self.db_channels = None


def run_ingestion(ctx: Context, cli: IngestionCLI) -> None:
    """Runs the ingestion process."""
    assert cli.reader is not None
    assert cli.db_channels is not None
    assert cli.client is not None
    assert cli.start is not None
    if cli.reader.type() == ReaderType.Row:
        engine = RowIngestionEngine(cli.client, cli.reader, cli.db_channels, cli.start)
    else:
        raise NotImplementedError("Only row ingestion is supported at this time.")
    ctx.console.info("Starting ingestion process...")
    engine.run()


def initialize_reader(
    ctx: Context,
    cli: IngestionCLI,
) -> str | None:
    ctx.console.info("Welcome to the Synnax ingestion CLI! Let's get started.")
    if cli.path is None:
        ctx.console.info("Please select a file to ingest.")
        cli.path = prompt_file(ctx)
    if cli.path is None:
        return None
    cli.reader = cli.factory.new_reader(cli.path)
    return "connect_client"


def _connect_client(ctx: Context, cli: IngestionCLI) -> str | None:
    """Prompts the user to connect to a Synnax client."""
    cli.client = connect_client(ctx)
    return "ingest_all" if cli.client else None


def ingest_all(ctx: Context, cli: IngestionCLI) -> str | None:
    """Prompts the user to ingest all channels."""
    assert cli.reader is not None
    if ctx.console.confirm("Would you like to ingest all channels?", default=True):
        cli.filtered_channels = cli.reader.channels()
        return "validate_channels_exist"
    else:
        return "channels_to_ingest"


def channels_to_ingest(ctx: Context, cli: IngestionCLI) -> str | None:
    """Prompts the user to select channels to ingest."""
    assert cli.reader is not None
    ctx.console.info("Which channels would you like to ingest?")
    channels = cli.reader.channels()
    grouped = prompt_group_channel_names(ctx, [ch.name for ch in channels])
    if grouped is None or len(grouped) == 0:
        return None
    all_names = [v for l in grouped.values() for v in l]
    cli.filtered_channels = [ch for ch in channels if ch.name in all_names]
    return "validate_channels_exist"


def validate_data_types(ctx: Context, cli: IngestionCLI) -> str | None:
    """Does an optimistic check on the first sample of each channel.  This isn't error
    prone, but checking every sample in a large file would be too slow.
    """
    assert cli.db_channels is not None
    assert cli.reader is not None

    cli.reader.set_chunk_size(1)
    cli.reader.seek_first()

    first = cli.reader.read()

    for ch in cli.db_channels:
        samples = first[ch.name]
        if len(samples) == 0:
            ctx.console.warn(f"Channel {ch.name} has no samples")

        samples_type, ch_type = samples.to_numpy().dtype, ch.data_type.numpy_type
        if samples.to_numpy().dtype != ch.data_type.numpy_type:
            ctx.console.error(
                f"""Unexpected data type for channel {ch.name}.  First sample is of
                    type {samples_type}, but channel  has  data type{ch_type}"""
            )
            return None

    return "validate_start_time"


def validate_channels_exist(ctx: Context, cli: IngestionCLI) -> str | None:
    """Validates that all channels in the file exist in the database. If not, prompts
    the user to create them.
    """
    assert cli.filtered_channels is not None
    assert cli.client is not None

    ctx.console.info("Validating that channels exist...")
    cli.not_found = []
    cli.db_channels = []
    for channel in cli.filtered_channels:
        ch = maybe_select_channel(
            ctx,
            cli.client.channel.filter(names=[channel.name]),
            channel.name,
        )
        if ch is None:
            cli.not_found.append(channel)
        else:
            cli.db_channels.append(ch)

    if len(cli.not_found) > 0:
        ctx.console.info("The following channels were not found in the database:")
        channel_name_table(ctx, [ch.name for ch in cli.not_found])
        if not ctx.console.confirm("Would you like to create them?"):
            return None
        return "create_channels"

    return "validate_data_types"


def validate_start_time(ctx: Context, cli: IngestionCLI) -> str | None:
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
        _start = ctx.console.ask_int(
            """Please enter the start timestamp of the file as a
                            nanosecond UTC integer. If you'd like a converter,
                            use https://www.epochconverter.com/""",
            default=TimeStamp.now(),
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
    if not ctx.console.confirm("Is this correct?"):
        return None

    return "ingest"


def create_indexes(
    ctx: Context,
    cli: IngestionCLI,
    options: list[ChannelMeta],
) -> list[ChannelMeta]:
    """Prompts the user to create index channels."""
    assert cli.client is not None
    assert cli.db_channels is not None
    grouped = prompt_group_channel_names(ctx, [ch.name for ch in options])
    if grouped is None:
        return options
    names = [name for v in grouped.values() for name in v]
    for name in names:
        ch = cli.client.channel.create(name=name, is_index=True, data_type=TIMESTAMP)
        cli.db_channels.append(ch)
    return [ch for ch in options if ch.name not in names]


def assign_dt(
    ctx: Context,
    cli: IngestionCLI,
) -> dict[str, list[ChannelMeta]] | None:
    assert cli.not_found is not None

    grouped = {GROUP_ALL: cli.db_channels}
    if not ctx.console.confirm("Do all channels have the same data type?"):
        if not ctx.console.confirm("Can you group channels by data type?"):
            grouped = {v.name: [v] for v in cli.not_found}
        grouped = prompt_group_channel_names(ctx, [ch.name for ch in cli.not_found])
        if grouped is None or len(grouped) == 0:
            return None
        grouped = {
            k: [ch for ch in cli.not_found if ch.name in v] for k, v in grouped.items()
        }

    assigned = {}
    for key, group in grouped.items():
        if key != GROUP_ALL:
            ctx.console.info(f"Assigning data type to {key}")
        dt = prompt_data_type_select(ctx)
        assigned[dt] = group

    return assigned


def assign_idx(
    ctx: Context,
    cli: IngestionCLI,
) -> dict[Rate | str, list[ChannelMeta]] | None:
    """Prompts the user to assign an index/rate to the channels in the given
    group"""
    assert cli.client is not None
    assert cli.not_found is not None
    client = cli.client

    grouped = {GROUP_ALL: cli.not_found}
    if not ctx.console.confirm(
        "Do all non-indexed channels have the same data rate or index?"
    ):
        if not ctx.console.confirm("Can you group channels by data rate or index?"):
            grouped = {v.name: [v] for v in cli.not_found}
        grouped = prompt_group_channel_names(ctx, [ch.name for ch in cli.not_found])
        if grouped is None or len(grouped) == 0:
            return None
        grouped = {
            k: [ch for ch in cli.not_found if ch.name in v] for k, v in grouped.items()
        }

    def assign_to_group(key: str, group: list[ChannelMeta]):
        if key != GROUP_ALL:
            ctx.console.info(f"Assigning data rate or index to {key}")
        _choice = ctx.console.ask("Enter the name of an index or a data rate")
        assert _choice is not None
        if _choice.replace(".", "").isdigit():
            return Rate(float(_choice))
        else:
            # If the user entered a string, we have an index channel, and we
            # need to make sure that the string is a valid index.
            res = client.channel.filter(names=[_choice])
            idx = maybe_select_channel(ctx, res, _choice)
            if not idx:
                ctx.console.warn(f"Index channel with key {_choice} not found")
                if ctx.console.confirm("Try again?"):
                    return assign_to_group(key, group)
                return None
            return idx.key

    assigned: dict[Rate | str, list[ChannelMeta]] = dict()
    for key, group in grouped.items():
        idx = assign_to_group(key, group)
        if idx is None:
            return None
        assigned[idx] = group

    return assigned


def create_channels(ctx: Context, cli: IngestionCLI) -> str | None:
    assert cli.not_found is not None
    assert cli.client is not None
    assert cli.db_channels is not None

    if ctx.console.confirm("Are any channels indexed (e.g. timestamps)?"):
        cli.not_found = create_indexes(ctx, cli, cli.not_found)

    idx_grouped, dt_grouped = assign_idx(ctx, cli), assign_dt(ctx, cli)
    if idx_grouped is None or dt_grouped is None:
        return None

    to_create = []
    for rate_or_index, channels in idx_grouped.items():
        is_rate = isinstance(rate_or_index, Rate)
        for ch in channels:
            to_create.append(
                Channel(
                    name=ch.name,
                    is_index=False,
                    index="" if is_rate else rate_or_index,
                    rate=rate_or_index if is_rate else 0,
                    data_type=[dt for dt, chs in dt_grouped.items() if ch in chs][0],
                )
            )

    cli.db_channels.extend(cli.client.channel.create_many(to_create))

    return "validate_data_types"
