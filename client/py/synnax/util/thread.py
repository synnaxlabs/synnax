#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from threading import Thread
from asyncio import events, tasks


def _cancel_all_tasks(loop):
    to_cancel = tasks.all_tasks(loop)
    if not to_cancel:
        return

    for task in to_cancel:
        task.cancel()

    loop.run_until_complete(tasks.gather(*to_cancel, return_exceptions=True))

    for task in to_cancel:
        if task.cancelled():
            continue
        if task.exception() is not None:
            loop.call_exception_handler(
                {
                    "message": "unhandled exception during asyncio.run() shutdown",
                    "exception": task.exception(),
                    "task": task,
                }
            )


class AsyncThread(Thread):
    loop: events.AbstractEventLoop

    def __init__(self):
        super().__init__()

    def run(self):
        self.loop = events.new_event_loop()
        try:
            events.set_event_loop(self.loop)
            self.loop.run_until_complete(self.run_async())
        finally:
            try:
                _cancel_all_tasks(self.loop)
                self.loop.run_until_complete(self.loop.shutdown_asyncgens())
                self.loop.run_until_complete(self.loop.shutdown_default_executor())
            finally:
                events.set_event_loop(None)
                self.loop.close()

    async def run_async(self):
        ...
