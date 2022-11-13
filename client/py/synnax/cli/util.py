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
    reader.set_chunk_size(100000000)
    df = reader.read()
    converted = convert_time_units(df[ch.name], curr, desired)
    df[ch.name] = converted.astype(np.int64)
    w = io_factory.new_writer(reader.path().parent / "converted.csv")
    w.write(df)
