#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.cli.console import Console, Sugared


class Context:
    console: Sugared

    def __init__(self, *, console: Console, prompt_enabled: bool = True):
        self.console = Sugared(prompt=console, print=console, enabled=prompt_enabled)
