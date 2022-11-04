from .flow import Context

from pathlib import Path


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
