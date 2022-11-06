import numpy as np

import synnax
from .channel import (
    maybe_select_channel,
    channel_name_table,
)
from .connect import connect_client, prompt_client_options, load_config_options
from .console.rich import RichConsole
from .file import prompt_file
from .select import select_from_table
from .suggest import determine_suggestions
from .telem import prompt_data_type_select
from .. import Synnax
from .flow import Context, Flow
from synnax.io import Factory
from ..io.reader import Reader, ChannelMeta
from ..telem import INT64, Rate
from ..cli.console.channel import prompt_group_channel_names


class IngestionCLI:
    factory: Factory
    reader: Reader | None
    client: Synnax | None
    filtered_channels: list[ChannelMeta] | None
    not_found: list[ChannelMeta] | None
    db_channels: list[synnax.Channel] | None

    def __init__(self, factory: Factory):
        self.factory = factory
        self.reader = None
        self.client = None
        self.filtered_channels = None
        self.not_found = None
        self.db_channels = None


def ingestion():
    flow = Flow(Context(console=RichConsole()))
    flow.add("initialize_reader", initialize_reader)
    flow.add("connect_client", _connect_client)
    flow.add("ingest_all", ingest_all)
    flow.add("channels_to_ingest", channels_to_ingest)
    flow.add("validate_channels", validate_channels)
    flow.add("create_channels", create_channels)
    flow.add("ingest", run_ingestion)
    flow.run(IngestionCLI(synnax.io.Factory()), "initialize_reader")


def initialize_reader(
    ctx: Context,
    cli: IngestionCLI,
) -> str | None:
    ctx.console.info("Welcome to the Synnax ingestion CLI! Let's get started.")
    ctx.console.info("Please select a file to ingest.")
    path = prompt_file(ctx)
    if path is None:
        return None
    cli.reader = cli.factory.new_reader(path)
    return "connect_client"


def _connect_client(ctx: Context, cli: IngestionCLI) -> str | None:
    """Prompts the user to connect to a Synnax client.
    """
    opts = load_config_options(ctx)
    if opts is None:
        opts = prompt_client_options(ctx)
    else:
        ctx.console.info("Using saved credentials.")
    cli.client = connect_client(ctx, opts)
    return "ingest_all" if cli.client else None


def ingest_all(ctx: Context, cli: IngestionCLI) -> str | None:
    """Prompts the user to ingest all channels.
    """
    if ctx.console.confirm("Would you like to ingest all channels?", default=True):
        cli.filtered_channels = cli.reader.channels()
        return "validate_channels"
    else:
        return "channels_to_ingest"


def channels_to_ingest(ctx: Context, cli: IngestionCLI) -> str | None:
    """Prompts the user to select channels to ingest.
    """
    ctx.console.info("Which channels would you like to ingest?")
    grouped = prompt_group_channel_names(ctx, [ch.name for ch in cli.reader.channels()])
    if grouped is None or len(grouped) == 0:
        return None
    cli.filtered_channels = [v for _, v in grouped.items()]
    return "validate_channels"


def validate_channels(ctx: Context, cli: IngestionCLI) -> str | None:
    """Validates that all channels in the file exist in the database. If not, prompts
    the user to create them.
    """
    ctx.console.info("Validating channels in file...")
    cli.not_found = []
    cli.db_channels = []
    for channel in cli.filtered_channels:
        ch = maybe_select_channel(
            ctx,
            cli.client.channel.retrieve_by_name(channel.name),
            channel.name,
        )
        if not ch:
            cli.not_found.append(channel)
        else:
            cli.db_channels.append(ch)

    if len(cli.not_found) > 0:
        ctx.console.info("The following channels were not found in the database:")
        channel_name_table(ctx, [ch.name for ch in cli.not_found])
        if not ctx.console.confirm("Would you like to create them?"):
            return None
        return "create_channels"

    return "ingest"


def create_indexes(
    ctx: Context,
    cli: IngestionCLI,
    options: list[ChannelMeta],
) -> list[ChannelMeta]:
    """Prompts the user to create index channels.
    """
    grouped = prompt_group_channel_names(ctx, [ch.name for ch in options])
    names = [name for v in grouped.values() for name in v]
    for name in names:
        cli.client.channel.create(name=name, is_index=True, data_type=INT64)
    return [ch for ch in options if ch.name not in names]


def group_by_idx(
    ctx: Context,
    options: list[ChannelMeta],
) -> dict[str, list[ChannelMeta]] | None:
    """Prompts the user to group channels by their index/rate
    """
    if not ctx.console.confirm(
        "Do all non-indexed channels have the same data rate or index?"
    ):
        if not ctx.console.confirm("Can you group channels by data rate or index?"):
            return {v.name: v for v in options}
        grouped = prompt_group_channel_names(ctx, [ch.name for ch in options])
        if grouped is None or len(grouped) == 0:
            return None
        return {k: v for k, v in grouped.items()}
    return {"__all__": options}


def create_channels(ctx: Context, cli: IngestionCLI) -> str | None:
    if ctx.console.confirm("Are any channels indexed (e.g. timestamps)?"):
        cli.not_found = create_indexes(ctx, cli, cli.not_found)

    idx_grouped = group_by_idx(ctx, cli.not_found)
    if idx_grouped is None:
        return None

    # First thing we must try to do is get the set of all data rates
    # or indexes that are being used in db_channels

    assigned_dr_idx = {}
    for key, group in idx_grouped.items():
        if key != "__all__":
            ctx.console.info(f"Assigning data rate or index to {key}")
        _choice = ctx.console.ask("Enter the name of an index or a data rate")
        if _choice.replace(".", "").isdigit():
            assigned_dr_idx[Rate(float(_choice))] = group
        else:
            # if the user entered a string, we have an index channel, and we
            #  need to make sure that the string is a valid index. Look first
            # in not_found then try to query
            res = cli.client.channel.retrieve_by_name(_choice)
            idx = maybe_select_channel(ctx, res, _choice)
            if not idx:
                ctx.console.ask(f"No channel found for index {_choice}")
                return None
            assigned_dr_idx[idx.key] = group

    dt_grouped = {}
    if not ctx.console.confirm("Do all of these channels have the same data type?"):
        if ctx.console.confirm("Can you group them by data type?"):
            grouped = prompt_group_channel_names(ctx, [ch.name for ch in cli.not_found])
            if grouped is None or len(grouped) == 0:
                return None
            dt_grouped = {k: v for k, v in grouped.items()}
        else:
            dt_grouped = {v: v for v in cli.not_found}
    else:
        dt_grouped["__all__"] = cli.not_found

    assigned_dts = {}
    for key, group in dt_grouped.items():
        if key != "__all__":
            ctx.console.info(f"Assigning data type to {key}")
        dt = prompt_data_type_select(ctx)
        assigned_dts[dt] = group

    for rate_or_index, channels in assigned_dr_idx.items():
        # if the rate_or_index is a string, it is an index channel
        index = ""
        rate: Rate = Rate(0)
        if isinstance(rate_or_index, Rate):
            rate = rate_or_index
        else:
            index = rate_or_index

        for channel in channels:
            # find the dt in which the channel exists
            dt = None
            for k, v in assigned_dts.items():
                if channel in v:
                    dt = k
                    break

            ch = cli.client.channel.create(
                name=channel.name,
                is_index=False,
                index=index,
                rate=rate,
                data_type=dt
            )
            cli.db_channels.append(ch)

    return "ingest"

def run_ingestion(ctx: Context, cli: IngestionCLI) -> None:
    """Runs the ingestion process.
    """
    ctx.console.info("Starting ingestion process...")
    data = cli.reader.read()
    # gseCh = [ch for ch in cli.db_channels if ch.name == "gse.timestamp (ul)"][0]
    # w = cli.client.data.new_writer([gseCh.key])
    # d = data["gse.timestamp (ul)"]
    # print(d.to_numpy(dtype=np.int64))
    # w.write(gseCh.key, d[0], d.to_numpy(dtype=np.int64))
    # w.close()
    w = cli.client.data.new_writer([ch.key for ch in cli.db_channels])
    for label, series in data.items():
        ch = [ch for ch in cli.db_channels if ch.name == label][0]
        w.write(ch.key, 0, series.to_numpy(dtype=np.float64))
    w.close()


