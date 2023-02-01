import click

from .ingest import ingest
from .login import login
from .ts_convert import pure_tsconvert


@click.group()
def synnax():
    ...


synnax.add_command(ingest)
synnax.add_command(login)
synnax.add_command(pure_tsconvert)
