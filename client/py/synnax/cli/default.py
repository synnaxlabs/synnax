#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.cli.console import RichConsole
from synnax.cli.flow import Context


def context(prompt_enabled: bool = True) -> Context:
    """Returns a new Context."""
    return Context(console=RichConsole(), prompt_enabled=prompt_enabled)
