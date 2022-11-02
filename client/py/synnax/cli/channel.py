import channels as channels

from synnax import Channel
from synnax.cli.flow import Context


def channel_table(
    ctx: Context,
    channels: list[Channel],
):
    ctx.console.table(
        columns=["option", "name", "key", "node_id", "index", "data_type", "data_rate"],
        rows=[{"option": i + 1, **ch.dict()} for i, ch in enumerate(channels)],
    )


def prompt_select_channel(
    ctx: Context,
    channels: list[Channel],
    allow_none: bool = False,
) -> Channel | None:
    ctx.console.info(f"Select a channel from the following list:")
    channel_table(ctx, channels)
    choices = [str(i + 1) for i in range(len(channels))]
    if allow_none:
        ctx.console.info("Enter '0' to select no channel.")
        choices.append("0")
    i = ctx.console.ask("Select an option #", choices=choices)
    if i == "0":
        return None
    return channels[int(i) - 1]


def prompt_group_channel_names(
    ctx: Context,
    group_names: list[str]
) -> dict[str, list[str]] | None:
    print("""You can enter 'all' for all channels or a comma-separated list of:
            "1)  names (e.g. 'channel1, channel2, channel3')
            "2) Channel indices (e.g. '1, 2, 3')
            "3) A pattern to match (e.g. 'channel*, sensor*')
            "4) A combination of the above (e.g. '1, 2, channel3, my_dog*')
        """)
    res = ctx.console.ask("Channels")
    return group_channel_names(group_names, res.split(","))


def group_channel_names(
    ctx: Context,
    all_names: list[str],
    matchers: list[str],
):
    grouped = {}
    for entry in matchers:
        entry = entry.strip()
        channels = []
        if entry.isdigit():
            index = int(entry)
            if index < 0 or index >= len(all_names):
                ctx.console.error(f"Invalid channel index: {index}[/]")
                skip = ctx.console.confirm("Skip?")
                if not skip:
                    return None
                continue
            channels.append(all_names[index])
        else:
            found = False
            for channel in all_names:
                print(entry, channel)
                if re.match(entry, channel):
                    channels.append(channel)
                    found = True
            if not found:
                ctx.console.error(f"[red]No channels found matching {entry}[/]")
                if not ctx.console.confirm("Skip?"):
                    return None
        grouped[entry] = channels

    return grouped
