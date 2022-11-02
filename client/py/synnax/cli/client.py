import dataclasses
import pathlib
import re
from typing import Callable

from freighter.exceptions import Unreachable
from rich import print
from rich.prompt import Prompt, IntPrompt, Confirm
from rich.table import Table

import synnax
from .console import Console
from .. import Synnax, AuthError
from synnax.io import Factory
from ..io.reader import Reader, ChannelMeta
from ..telem import DATA_TYPES, INT64
from ..cli.console.channel import prompt_select_channel


class IngestionRequest:
    console: Console
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


def prompt_file(req: IngestionRequest) -> str | None:
    req.console.info("Enter the path to the file you want to parse: \n [/]")
    fp = req.console.ask("Filepath", default="./synnax/valid.csv")
    req.reader = req.factory.new_reader(pathlib.Path(fp))
    req.console.success(
        f"File opened successfully! Using a {req.reader.extensions()[0]} parser.[/]")
    return "prompt_client"


def prompt_client(req: IngestionRequest) -> str | None:
    print("Enter your Synnax connection parameters:")
    hostname = Prompt.ask("[#3774d0]Host[/]", default="localhost")
    port = IntPrompt.ask("[#3774d0]Port[/]", default=9090)
    username = Prompt.ask("[#3774d0]Username[/]", default="synnax")
    password = Prompt.ask("[#3774d0]Password[/]", password=True, default="seldon")

    try:
        req.client = Synnax(
            host=hostname,
            port=port,
            username=username,
            password=password,
        )
    except Unreachable:
        print(f"[red]Cannot reach Synnax server at {hostname}:{port}[/]")
        return None
    except AuthError:
        print(f"[red]Invalid credentials[/]")
        return None
    except Exception as e:
        print(f"[red]An error occurred: {e}[/]")
        return None

    print("[green]Connection successful![/]")

    return "prompt_ingest_all"


def prompt_ingest_all(req: IngestionRequest) -> str | None:
    ingest_all = Confirm.ask("Would you like to ingest all channels?")
    if ingest_all:
        req.filtered_channels = req.reader.channels()
        return "validate_channels"
    else:
        return "prompt_channels_to_ingest"


def prompt_channels_to_ingest(req: IngestionRequest) -> str | None:
    print("[#3774d0]Which channels would you like to ingest?[/]")
    print(req.reader.channels())
    grouped = prompt_group_channels([ch.key for ch in req.reader.channels()])
    if grouped is None or len(grouped) == 0:
        return None
    req.filtered_channels = [v for _, v in grouped.items()]
    return "validate_channels"


def validate_channels(req: IngestionRequest) -> str | None:
    print("[#3774d0]Checking if channels exist...[/]")
    req.not_found = []
    req.db_channels = []
    for channel in req.filtered_channels:
        res = req.client.channel.retrieve_by_name([channel.key])
        if len(res) == 0:
            req.not_found.append(channel)
        elif len(res) > 1:
            print(f"[red]Multiple channels found for {channel.key}[/]")
            selected = prompt_select_channel(res, True)
            if selected is None:
                return None
            req.db_channels.append(selected)
        else:
            req.db_channels.append(res[0])

    if len(req.not_found) > 0:
        print("[#3774d0]The following channels were not found:[/]")
        t = Table(show_header=True, header_style="bold magenta")
        t.add_column("Channel Name", style="cyan")

        for channel in req.not_found:
            t.add_row(channel.key)

        print(t)

        if not Confirm.ask("Would you like to create them?"):
            return None

        return "prompt_create_channels"

    return "ingest"


def prompt_create_channels(req: IngestionRequest) -> str | None:
    if Confirm.ask(
        "[#3774d0]Are any of these channels indexed (are they a timestamp column)?"):
        grouped = prompt_group_channels([v.key for v in req.not_found])
        for k, v in grouped.items():
            for name in v:
                req.client.channel.create(
                    name=name,
                    is_index=True,
                    data_type=INT64
                )
            req.not_found = [ch for ch in req.not_found if ch not in v]

    dr_grouped = {}
    if not Confirm.ask(
        "[#3774d0]Do all of these channels have the same data rate or index?"):
        can_group = Confirm.ask("[#377d40]Can you group them by data rate or index?[/]")
        if can_group:
            grouped = prompt_group_channels(req.not_found)
            if grouped is None or len(grouped) == 0:
                return None
            dr_grouped = {k: v for k, v in grouped.items()}
        else:
            dr_grouped = {v: v for v in req.not_found}
    else:
        dr_grouped["__all__"] = req.not_found

    # First thing we must try to do is get the set of all data rates
    # or indexes that are being used in db_channels
    suggested_indexes_or_rates = set()
    for ch in req.db_channels:
        if ch.index is not None:
            suggested_indexes_or_rates.add(ch.index)
        else:
            suggested_indexes_or_rates.add(ch.data_rate)
    suggested_indexes_or_rates = list(suggested_indexes_or_rates)

    assigned_drs = {}
    idx_keys = {}
    for key, group in dr_grouped.items():
        if key != "__all__":
            print(f"[#3774d0]Assigning data rate for channels for {key}[/]")
        if len(suggested_indexes_or_rates) >= 0:
            print("[#3774d0]The following indexes or data rates are already in use:[/]")
            # make an indexed table of option #s
            t = Table(show_header=True, header_style="bold magenta")
            t.add_column("Option #", justify="right", style="dim", no_wrap=True)
            t.add_column("Index or Data Rate", style="cyan")
            for i, index_or_rate in enumerate(suggested_indexes_or_rates):
                t.add_row(str(i + 1), str(index_or_rate))
            print(
                "[#3774d0]You may select one of these or enter 0 to use a custom index or data rate[/]")
            choice = Prompt.ask("Index or Data Rate",
                                choices=["0", *[str(i + 1) for i in range(
                                    len(suggested_indexes_or_rates) + 1)]],
                                )
            custom = False
            if choice == "0":
                choice = Prompt.ask("Index or Data Rate")
                custom = True

            parsed_choice = choice
            if choice.replace(".", "", 1).isdigit():
                if custom:
                    parsed_choice = float(choice)
                else:
                    parsed_choice = suggested_indexes_or_rates[int(choice) - 1]
            else:
                # if the user entered a string, we have an index channel, and we need
                # to make sure that the string is a valid index. Look first in not_found
                # then try to query
                res = req.client.channel.retrieve_by_name(parsed_choice)
                if len(res) == 0:
                    print(f"[red]No channel found for index {parsed_choice}[/]")
                    return None
                elif len(res) > 1:
                    print(f"[red]Multiple channels found for index {parsed_choice}[/]")
                    prompt_select_channel(res, True)
                    idx_keys[key] = parsed_choice
                else:
                    idx_keys[key] = res[0]

            assigned_drs[parsed_choice] = group

    dt_grouped = {}
    if not Confirm.ask(
        "[#3774d0]Do all of these channels have the same data type or index?"):
        can_group = Confirm.ask("[#377d40]Can you group them by data type?[/]")
        if can_group:
            grouped = prompt_group_channels(req.not_found)
            if grouped is None or len(grouped) == 0:
                return None
            dt_grouped = {k: v for k, v in grouped.items()}
        else:
            dt_grouped = {v: v for v in req.not_found}
    else:
        dt_grouped["__all__"] = req.not_found

    assigned_dts = {}
    for key, group in dt_grouped.items():
        dt_strings = [dt.string() for dt in DATA_TYPES]
        t = Table(show_header=True, header_style="bold magenta")
        t.add_column("Option #", justify="right", style="dim", no_wrap=True)
        t.add_column("Data Type", style="cyan")
        for i, dt in enumerate(dt_strings):
            t.add_row(str(i + 1), dt)
        print(t)

        c = Prompt.ask("What data types would you like to use for these channels?",
                       choices=[str(i + 1) for i in range(len(dt_strings))])

        dt = DATA_TYPES[int(c) - 1]
        assigned_dts[dt] = group

    for rate_or_index, channels in assigned_drs.items():
        # if the rate_or_index is a string, it is an index channel
        index = ""
        rate: float = 0
        if type(rate_or_index) == str:
            index = idx_keys[rate_or_index]
        else:
            rate = rate_or_index

        for channel in channels:
            # find the dt in which the channel exists
            dt = None
            for k, v in assigned_dts.items():
                if channel in v:
                    dt = k
                    break

            # create the channel
            print(dt)
            ch = req.client.channel.create(
                name=channel.key,
                is_index=False,
                index=index,
                rate=rate,
                data_type=dt
            )
            req.db_channels.append(ch)

    return None


class Workflow:
    steps: dict[str, Callable[[IngestionRequest], str | None]]

    def __init__(self):
        self.steps = {}

    def register(self, name: str, step: Callable[[IngestionRequest], str | None]):
        self.steps[name] = step

    def run(self, request: IngestionRequest, root: str):
        root_step = self.steps[root]
        self._run(root_step, request)

    def _run(self,
             step: Callable[[IngestionRequest], str | None],
             request: IngestionRequest,
             ):
        next_step = step(request)
        if next_step is not None:
            self._run(self.steps[next_step], request)


def run():
    f = Factory()
    w = Workflow()
    r = IngestionRequest(factory=f)
    w.register("prompt_file", prompt_file)
    w.register("prompt_client", prompt_client)
    w.register("prompt_ingest_all", prompt_ingest_all)
    w.register("validate_channels", validate_channels)
    w.register("prompt_create_channels", prompt_create_channels)
    w.register("prompt_ingest", lambda req: print(req))
    w.run(r, "prompt_file")
