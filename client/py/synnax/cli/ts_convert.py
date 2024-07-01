#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path
from rich.progress import (
    BarColumn,
    Progress,
    TaskProgressColumn,
    TextColumn,
    TimeElapsedColumn,
)
import click
import time
from datetime import datetime

from synnax.cli import default
from synnax.cli.flow import Context
from synnax.cli.io import prompt_new_reader
from synnax.cli.telem import ask_time_units_select
from synnax.io import BaseReader
from synnax.io.factory import IO_FACTORY
from synnax.telem import TimeSpanUnits, convert_time_units

OUTPUT_PATH_ARG = "--output-path"
OUTPUT_PATH_ARG_SHORT = "-o"
INPUT_CHANNEL_ARG = "--input-channel"
INPUT_CHANNEL_ARG_SHORT = "-ic"
OUTPUT_CHANNEL_ARG = "--output-channel"
OUTPUT_CHANNEL_ARG_SHORT = "-oc"
INPUT_PRECISION_ARG = "--input-precision"
INPUT_PRECISION_ARG_SHORT = "-ip"
OUTPUT_PRECISION_ARG = "--output-precision"
OUTPUT_PRECISION_ARG_SHORT = "-op"


@click.command()
@click.argument(
    "input_path",
    required=False,
)
@click.option(
    OUTPUT_PATH_ARG,
    OUTPUT_PATH_ARG_SHORT,
    "output_path",
    required=False,
    help="The path to save the converted file",
)
@click.option(
    INPUT_PRECISION_ARG,
    INPUT_PRECISION_ARG_SHORT,
    "input_precision",
    required=False,
    help="The current precision of the time units",
)
@click.option(
    OUTPUT_PRECISION_ARG,
    OUTPUT_PRECISION_ARG_SHORT,
    "output_precision",
    required=False,
    help="The desired precision of the time units",
)
@click.option(
    INPUT_CHANNEL_ARG,
    INPUT_CHANNEL_ARG_SHORT,
    "input_channel",
    required=False,
    help="The channel to convert",
)
@click.option(
    OUTPUT_CHANNEL_ARG,
    OUTPUT_CHANNEL_ARG_SHORT,
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
    pure_tsconvert(
        input_path,
        output_path,
        input_channel,
        output_channel,
        input_precision,
        output_precision,
        default.context(prompt_enabled=prompt),
    )


def pure_tsconvert(
    input_path: Path | str | None,
    output_path: Path | str | None,
    input_channel: str | None,
    output_channel: str | None,
    input_precision: TimeSpanUnits | None,
    output_precision: TimeSpanUnits | None,
    ctx: Context,
) -> None:
    reader = prompt_new_reader(ctx, input_path)
    if reader is None:
        return
    input_path = reader.path()

    input_channel = ask_channel_and_check_exists(
        ctx,
        reader,
        question="Which channel would you like to convert?",
        arg_name=INPUT_CHANNEL_ARG,
        arg=input_channel,
    )

    input_precision = ask_time_units_select(
        ctx,
        question="What is the current precision?",
        arg_name=INPUT_PRECISION_ARG,
        arg=input_precision,
    )

    output_precision = ask_time_units_select(
        ctx,
        question="What is the desired precision?",
        arg_name=OUTPUT_PRECISION_ARG,
        arg=output_precision,
    )

    output_channel = ctx.console.ask(
        "What would you like to name the output channel?",
        default=input_channel,
        arg=output_channel,
        arg_name=OUTPUT_CHANNEL_ARG,
    )

    output_path = Path(
        ctx.console.ask(
            "Where would you like to save the converted data?",
            default=str(
                input_path.parent / f"{input_path.stem}_converted{input_path.suffix}"
            ),
            arg=str(output_path) if output_path is not None else None,
            arg_name=OUTPUT_PATH_ARG,
        )
    )

    writer = IO_FACTORY.open_writer(output_path)

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
                t0 = datetime.now()
                converted = convert_time_units(
                    chunk[input_channel], input_precision, output_precision
                )
                chunk[output_channel] = converted
                writer.write(chunk)
                tp = chunk.size / (datetime.now() - t0).total_seconds()
                progress.update(task, advance=chunk.size, tp=int(tp))
    finally:
        reader.close()
        writer.close()


def ask_channel_and_check_exists(
    ctx: Context,
    reader: BaseReader,
    question="Enter a channel name",
    arg_name="channel",
    arg: str | None = None,
) -> str:
    _ch = ctx.console.ask(question, arg_name=arg_name, arg=arg)
    try:
        next(ch for ch in reader.channels() if ch.name == _ch)
    except StopIteration:
        ctx.console.error(f"Channel not found: {_ch}")
        if arg is None:
            return ask_channel_and_check_exists(ctx, reader, question, arg_name, arg)
        else:
            ctx.console.error(f"Channel not found: {_ch}")
    return _ch
