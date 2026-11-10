#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from pathlib import Path

import click
from rich.progress import (
    BarColumn,
    Progress,
    TaskProgressColumn,
    TextColumn,
    TimeElapsedColumn,
)

from synnax.cli import default
from synnax.cli.flow import Context
from synnax.cli.io import prompt_new_reader
from synnax.cli.telem import ask_time_units_select
from synnax.io import BaseReader
from synnax.io.factory import FACTORY
from synnax.telem import TimeSpanUnits, convert_time_units

_OUTPUT_PATH_ARG = "--output-path"
_OUTPUT_PATH_ARG_SHORT = "-o"
_INPUT_CHANNEL_ARG = "--input-channel"
_INPUT_CHANNEL_ARG_SHORT = "-ic"
_OUTPUT_CHANNEL_ARG = "--output-channel"
_OUTPUT_CHANNEL_ARG_SHORT = "-oc"
_INPUT_PRECISION_ARG = "--input-precision"
_INPUT_PRECISION_ARG_SHORT = "-ip"
_OUTPUT_PRECISION_ARG = "--output-precision"
_OUTPUT_PRECISION_ARG_SHORT = "-op"


@click.command()
@click.argument(
    "input_path",
    required=False,
)
@click.option(
    _OUTPUT_PATH_ARG,
    _OUTPUT_PATH_ARG_SHORT,
    "output_path",
    required=False,
    help="The path to save the converted file",
)
@click.option(
    _INPUT_PRECISION_ARG,
    _INPUT_PRECISION_ARG_SHORT,
    "input_precision",
    required=False,
    help="The current precision of the time units",
)
@click.option(
    _OUTPUT_PRECISION_ARG,
    _OUTPUT_PRECISION_ARG_SHORT,
    "output_precision",
    required=False,
    help="The desired precision of the time units",
)
@click.option(
    _INPUT_CHANNEL_ARG,
    _INPUT_CHANNEL_ARG_SHORT,
    "input_channel",
    required=False,
    help="The channel to convert",
)
@click.option(
    _OUTPUT_CHANNEL_ARG,
    _OUTPUT_CHANNEL_ARG_SHORT,
    "output_channel",
    required=False,
    help="The name of the output channel. Defaults to the name of the input channel.",
)
@click.option(
    "-p",
    "--prompt/--no-prompt",
    "prompt",
    help="Prompt the user for missing information",
    default=True,
)
def tsconvert(
    input_path: str | None,
    output_path: str | None,
    input_channel: str | None,
    output_channel: str | None,
    input_precision: TimeSpanUnits | None,
    output_precision: TimeSpanUnits | None,
    prompt: bool,
) -> None:
    """Converts the time units of a channel in a file.

    All arguments are optional. If not provided, the user will be prompted for
    the missing information.
    """
    _pure_tsconvert(
        input_path,
        output_path,
        input_channel,
        output_channel,
        input_precision,
        output_precision,
        default.context(prompt_enabled=prompt),
    )


def _pure_tsconvert(
    input_path: Path | str | None,
    output_path: Path | str | None,
    input_channel: str | None,
    output_channel: str | None,
    input_precision: TimeSpanUnits | None,
    output_precision: TimeSpanUnits | None,
    ctx: Context,
) -> None:
    reader = prompt_new_reader(ctx, path=input_path)
    if reader is None:
        return
    input_path = reader.path()

    input_channel = _ask_channel_and_check_exists(
        ctx,
        reader,
        question="Which channel would you like to convert?",
        arg_name=_INPUT_CHANNEL_ARG,
        arg=input_channel,
    )

    input_precision = ask_time_units_select(
        ctx,
        question="What is the current precision?",
        arg_name=_INPUT_PRECISION_ARG,
        arg=input_precision,
    )

    output_precision = ask_time_units_select(
        ctx,
        question="What is the desired precision?",
        arg_name=_OUTPUT_PRECISION_ARG,
        arg=output_precision,
    )

    output_channel = ctx.console.ask(
        "What would you like to name the output channel?",
        default=input_channel,
        arg=output_channel,
        arg_name=_OUTPUT_CHANNEL_ARG,
    )

    output_path_str = ctx.console.ask(
        "Where would you like to save the converted data?",
        default=str(
            input_path.parent / f"{input_path.stem}_converted{input_path.suffix}"
        ),
        arg=str(output_path) if output_path is not None else None,
        arg_name=_OUTPUT_PATH_ARG,
    )
    if output_path_str is None:
        raise ValueError("Output path is required")
    output_path = Path(output_path_str)

    writer = FACTORY.open_writer(output_path)

    reader.seek_first()
    try:
        with Progress(
            BarColumn(),
            TaskProgressColumn(),
            TextColumn("{task.completed} out of {task.total} samples"),
            TimeElapsedColumn(),
            TextColumn("{task.fields[tp]} samples/s"),
        ) as progress:
            task = progress.add_task("convert", total=reader.nsamples(), tp=0)
            for chunk in reader:
                t0 = time.perf_counter()
                converted = convert_time_units(
                    chunk[input_channel].to_numpy(),
                    _from=input_precision,
                    to=output_precision,
                )
                chunk[output_channel] = converted
                writer.write(chunk)
                tp = chunk.size / (time.perf_counter() - t0)
                progress.update(task, advance=chunk.size, tp=int(tp))
    finally:
        reader.close()
        writer.close()


def _ask_channel_and_check_exists(
    ctx: Context,
    reader: BaseReader,
    question: str = "Enter a channel name",
    arg_name: str = "channel",
    arg: str | None = None,
) -> str:
    _ch = ctx.console.ask(question, arg_name=arg_name, arg=arg)
    if _ch is None:
        raise ValueError("Channel name is required")
    try:
        next(ch for ch in reader.channels() if ch.name == _ch)
    except StopIteration:
        ctx.console.error(f"Channel not found: {_ch}")
        if arg is None:
            return _ask_channel_and_check_exists(ctx, reader, question, arg_name, arg)
        ctx.console.error(f"Channel not found: {_ch}")
    return _ch
