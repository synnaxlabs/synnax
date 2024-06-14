#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import fnmatch

from synnax.channel import Channel
from synnax.cli.console.sugared import AskKwargs
from synnax.cli.flow import Context


def channel_name_table(
    ctx: Context,
    names: list[str],
):
    """Creates a table containing names of the channels.

    :param ctx: The current flow context.
    :param names: The names of the channels.
    """
    ctx.console.table(
        columns=["name"],
        rows=[{"name": name} for name in names],
    )


def maybe_select_channel(
    ctx: Context,
    channels: list[Channel],
    param: str,
    **kwargs: AskKwargs[str],
) -> Channel | None:
    """Asks the user to select a channel if there are multiple channels available.

    :param ctx: The current flow context.
    :param channels:  The list of channels to prompt from.
    :returns: The selected channel or None if there are no channels.
    """
    if len(channels) == 0:
        return None
    if len(channels) > 1:
        ctx.console.error(f"Multiple channels found for {param}!")
        return select_channel(ctx, channels, **kwargs)
    return channels[0]


def select_channel(
    ctx: Context,
    channels: list[Channel],
    **kwargs: AskKwargs[str],
) -> Channel | None:
    """Prompts the user to select a channel from a list of channels.

    :param ctx: The current flow Context.
    :param channels: The list of channels to select from.
    :param default: The default channel to select.
    :param allow_none: Whether to allow the user to select None.
    :returns: The selected channel or None if there are no channels.
    """
    print(channels)
    _, i = ctx.console.select(
        type_=str,
        columns=["name", "key", "data_type", "index", "rate", "leaseholder"],
        rows=[c.dict() for c in channels],
        **kwargs,
    )
    return channels[i]


def prompt_group_channel_names(
    ctx: Context, options: list[str]
) -> dict[str, list[str]] | None:
    """Prompts the user to group channel names by providing a list of matchers.

    :param ctx: The current flow Context.
    :param options: The list of channel names to match against.
    :param options: The grouped dict. See "group_channel_names() for reference."
    """
    ctx.console.info(
        """You can enter 'all' for all channels or a comma-separated list of:
    1) Names (e.g. 'channel1, channel2, channel3')
    2) Channel indices (e.g. '1, 2, 3')
    3) A pattern to match (e.g. 'channel*, sensor*')
    """
    )
    return group_channel_names(ctx, options, ctx.console.ask("channels").split(","))


def group_channel_names(
    ctx: Context,
    options: list[str],
    matchers: list[str],
):
    """Groups channel names by matching them against a list of matchers.

    :param ctx: The current flow Context.
    :param options: The list of channel names to match against.
    :param matchers: The list of matchers to use. Each matcher can be:
    - A name of a channel (e.g. 'channel1')
    - An index of a channel (e.g. '1')
    - A pattern to match (e.g. 'channel*')
    :returns: A dict containing groups of channels organized by name, index, pattern.
    """
    grouped = {}
    for entry in matchers:
        entry = entry.strip()
        channels = list()
        if entry.isdigit():
            index = int(entry)
            if index < 0 or index >= len(options):
                ctx.console.error(f"Invalid channel index: {index}[/]")
                if not ctx.console.ask("Continue?", bool, default=True):
                    return None
                continue
            channels.append(options[index])
        else:
            for channel in options:
                if fnmatch.fnmatch(channel, entry) or channel == entry:
                    channels.append(channel)
            if len(channels) == 0:
                ctx.console.error(f"[red]No channels found matching {entry}[/]")
                if not ctx.console.ask("Continue?", bool, default=True):
                    return None
        grouped[entry] = channels

    return grouped
