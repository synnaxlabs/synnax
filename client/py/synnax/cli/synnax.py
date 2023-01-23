import click

from .ingest import ingest
from .login import login
from .util import tsconvert


@click.group()
def synnax():
    ...


synnax.add_command(ingest)
synnax.add_command(login)
synnax.add_command(tsconvert)
