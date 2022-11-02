from rich import print
from rich.table import Table
from rich.prompt import Prompt

from synnax import Channel


def prompt_select_channel(channels: list[Channel], allow_none: bool = False) -> Channel:
    print(f"[#3774d0]Select a channel from the following list:")
    t = Table(show_header=True, header_style="bold magenta")
    t.add_column("Option #", justify="right", style="dim", no_wrap=True)
    t.add_column("Channel Name", style="cyan")
    t.add_column("Channel Key", style="cyan")
    t.add_column("Node ID", style="cyan")
    t.add_column("Index", style="cyan")
    t.add_column("Data Type", style="cyan")
    t.add_column("Data Rate", style="cyan")

    for i, ch in enumerate(channels):
        t.add_row(str(i), ch.name, ch.key, str(ch.node_id), str(ch.index),
                  ch.data_type, str(ch.data_rate))

    print(t)
    choices = [str(i) for i in range(len(channels))]
    if allow_none:
        print(f"[#3774d0]Enter 'none' to select no channel.")
        choices.append("none")
    i = Prompt.ask("Select an option #", choices=choices)
    return channels[int(i)]
