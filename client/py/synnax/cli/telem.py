from .channel import select_from_table
from ..telem import DATA_TYPES, DataType
from .flow import Context


def prompt_data_type_select(ctx: Context, allow_none: bool = False) -> DataType | None:
    """Prompts the user to select a data type from a list of all available data
    types.

    :param ctx: The current flow Context.
    :param allow_none: Whether to allow the user to select None.
    """
    i = select_from_table(
        ctx,
        ["option", "data_type"],
        [{"data_type": name.string()} for name in DATA_TYPES],
        allow_none=allow_none,
    )
    print(i, DATA_TYPES[i])
    return DATA_TYPES[i] if i is not None else None
