from .flow import Context

from pathlib import Path

from ..io import IOFactory, RowReader

io_factory = IOFactory()


def prompt_file(ctx: Context) -> Path | None:
    """
    Prompts the user for a file path.
    """
    fp = Path(ctx.console.ask("File path"))
    if not fp.exists():
        ctx.console.error(f"File does not exist: {fp}")
        return None
    ctx.console.success(f"File found: {fp}")
    return fp


def prompt_new_reader(ctx: Context) -> RowReader | None:
    """
    Prompts the user for a file path and returns a new reader for that file.
    """
    fp = prompt_file(ctx)
    if fp is None:
        return None
    return io_factory.new_reader(fp)
