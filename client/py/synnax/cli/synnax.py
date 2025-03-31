#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import click

from .check_timing import check_timing
from .ingest import ingest
from .login import login
from .ts_convert import tsconvert


@click.group()
def synnax() -> None: ...


synnax.add_command(ingest)
synnax.add_command(login)
synnax.add_command(tsconvert)
synnax.add_command(check_timing)
