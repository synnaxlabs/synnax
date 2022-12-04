from .flow import Context


def select_from_table(
    ctx: Context,
    columns: list[str],
    rows: list[dict[str, any]],
    allow_none: bool = False,
    default: int = None,
) -> int | None:
    """Prompts the user to select a row.py from a table.

    :param ctx: The current flow Context.
    :param columns: The columns of the table.
    :param rows: The rows of the table.
    :param allow_none: Whether to allow the user to select nothing.
    :param default: The default option to select. If a default is provided,
    allow_none is ignored.
    :returns: The index of the selected row.py or None if nothing was selected.
    """
    ctx.console.table(columns=["option", *columns], rows=[
        {"option": str(i), **row} for i, row in enumerate(rows)
    ])
    choices = [str(i) for i in range(len(rows))]
    if allow_none and default is None:
        default = "None"
        ctx.console.info("Press enter to select nothing.")
    i = ctx.console.ask("Select an option #", choices=choices, default=default)
    return None if i is "None" else int(i)
