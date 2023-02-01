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
import numpy as np

from synnax.io import BaseReader
from synnax.io.factory import IO_FACTORY
from synnax.telem import convert_time_units
from synnax.cli.flow import Context
from synnax.cli.io import prompt_new_reader
from synnax.cli.telem import ask_time_units_select
from synnax.cli.args import if_none
from synnax.cli import default


@click.command()
@click.argument("path", required=False)
@click.option("-ip", "--input", required=False)
@click.option("-op", "--output", required=False)
@click.option("-o", "--out", required=False)
@click.option("-c", "--channel", required=False)
def tsconvert(
    path: str | None,
    out: str | None,
    channel: str | None,
    input: str | None,
    output: str | None,
) -> None:
    pure_tsconvert(path, out, channel, input, output)


def pure_tsconvert(
    path: Path | str | None,
    out: Path | str | None,
    channel: str | None,
    input: str | None,
    output: str | None,
    ctx: Context = default.context(),
) -> None:
    reader = prompt_new_reader(ctx, path)
    if reader is None:
        return

    channel = ask_channel_and_check_exists(
        ctx,
        reader,
        channel,
        "Which channel would you like to convert?",
    )
    if channel is None:
        return

    input = if_none(
        input,
        ask_time_units_select,
        ctx,
        question="What is the current precision?",
        required=True,
    )
    output = if_none(
        output,
        ask_time_units_select,
        ctx,
        question="What is the desired precision?",
        required=True,
    )
    out = if_none(
        out,
        ctx.console.ask,
        "Where would you like to save the converted data?",
        required=True,
    )
    assert output is not None and input is not None and out is not None
    writer = IO_FACTORY.new_writer(reader.path().parent / out)
    reader.seek_first()
    for chunk in reader:
        chunk[channel] = convert_time_units(chunk[channel], input, output).astype(
            np.int64
        )
        writer.write(chunk)


def ask_channel_and_check_exists(
    ctx: Context,
    reader: BaseReader,
    channel: str | None,
    question="Enter a channel name",
) -> str | None:
    assert reader is not None
    _ch = if_none(
        channel,
        ctx.console.ask,
        question,
        required=True,
    )
    try:
        next(ch for ch in reader.channels() if ch.name == _ch)
    except StopIteration:
        ctx.console.error(f"Channel not found: {_ch}")
        if channel is None:
            return ask_channel_and_check_exists(ctx, reader, channel, question)
    return _ch
