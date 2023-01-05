#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

from .console import RichConsole
from .flow import Context
from .io import prompt_new_reader, io_factory
from .telem import prompt_time_units_select
from ..telem import convert_time_units


def convert_timestamp_precision():
    ctx = Context(console=RichConsole())
    reader = prompt_new_reader(ctx)
    c = ctx.console.ask("Which channel would you like to convert?")
    channels = reader.channels()
    ch = next((ch for ch in channels if ch.name == c), None)
    if ch is None:
        ctx.console.error(f"Channel not found: {c}")
        return

    ctx.console.info("What is the current precision?")
    curr = prompt_time_units_select(ctx)
    ctx.console.info("What is the desired precision?")
    desired = prompt_time_units_select(ctx)
    path = ctx.console.ask("Where would you like to save the converted data?")
    writer = io_factory.new_writer(reader.path().parent / path)
    reader.set_chunk_size(25000)
    while True:
        try:
            chunk = reader.read()
        except StopIteration:
            break
        chunk[ch.name] = convert_time_units(chunk[ch.name], curr, desired).astype(
            np.int64
        )
        writer.write(chunk)
