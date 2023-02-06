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

from synnax.io import BaseReader
from synnax.io.factory import IO_FACTORY
from synnax.telem import convert_time_units
from synnax.cli.flow import Context
from synnax.cli.io import prompt_new_reader
from synnax.cli.telem import ask_time_units_select
from synnax.cli import default

OUTPUT_PATH_ARG = "--output-path"
INPUT_PATH_ARG = "--input-path"
INPUT_CHANNEL_ARG = "--input-channel"
OUTPUT_CHANNEL_ARG = "--output-channel"
INPUT_PRECISION_ARG = "--input-precision"
OUTPUT_PRECISION_ARG = "--output-precision"

@click.command()
@click.argument(
    "input_path", 
    required=False, 
)
@click.option(
    "-o",
    "--output-path",
    "output_path", 
    required=False,
    help="The path to save the converted file",
)
@click.option(
    "-ip", 
    "--input-precision",
    "input_precision", 
    required=False,
    help="The current precision of the time units",
)
@click.option(
    "-op", 
    "--output-precision",
    "output_precision", 
    required=False,
    help="The desired precision of the time units",
)
@click.option(
    "-ic", 
    "--input-channel",
    "input_channel", 
    required=False,
    help="The channel to convert",
)
@click.option(
    "-oc", 
    "--output-channel",
    "output_channel", 
    required=False,
    help="The name of the output channel",
)
@click.option(
    "-p",
    "--prompt/--no-prompt",
    "prompt",
    help="Prompt the user for missing information",
    default=True
)
def tsconvert(
    input_path: str | None,
    output_path: str | None,
    input_channel: str | None,
    output_channel: str | None,
    input_precision: str | None,
    output_precision: str | None,
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
        default.context(prompt=prompt),
    )


def pure_tsconvert(
    input_path: Path | str | None,
    output_path: Path | str | None,
    input_channel: str | None,
    output_channel: str | None,
    input_precision: str | None,
    output_precision: str | None,
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
        value=input_precision,
        arg_name=INPUT_PRECISION_ARG,
        arg=input_precision,
    )

    output_precision = ask_time_units_select(
        ctx,
        question="What is the desired precision?",
        value=output_precision
    )

    output_channel = ctx.console.ask(
        "What would you like to name the output channel?",
        default=input_channel,
    )

    output_path = Path(ctx.console.ask(
        "Where would you like to save the converted data?",
        if_none=str(output_path) if output_path is not None else None,
        default=str(input_path.parent / f"{input_path.stem}_converted{input_path.suffix}"),
    ))

    writer = IO_FACTORY.new_writer(output_path)

    reader.seek_first()

    for chunk in reader:
        converted = convert_time_units(chunk[input_channel], input_precision, output_precision)
        chunk[output_channel] = converted
        writer.write(chunk)


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
