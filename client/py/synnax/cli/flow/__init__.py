#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Generic, TypeVar, Callable

from synnax.cli.console import Console

T = TypeVar("T")


class Context:
    console: Console

    def __init__(self, console: Console):
        self.console = console


class Flow(Generic[T]):
    steps: dict[str, Callable[[Context, T], str | None]]
    context: Context

    def __init__(self, ctx: Context):
        self.steps = {}
        self.context = ctx

    def add(self, name: str, step: T):
        self.steps[name] = step

    def run(self, req: T, root: str):
        root_step = self.steps[root]
        self._run(root_step, req)

    def _run(
        self,
        step: Callable[[T, Context], str | None],
        request: T,
    ):
        next_step = step(self.context, request)
        if next_step is not None:
            self._run(self.steps[next_step], request)
