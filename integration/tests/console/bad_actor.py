#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from console.case import ConsoleCase


class BadActor(ConsoleCase):
    """
    Attempt to delete channels that are being controlled
    by another process. Test will PASS if channels could
    not be deleted.
    """

    def run(self) -> None:
        """
        Test the "create a channel" modal for all data types
        """
        console = self.console
        client = self.client

        channels_to_delete = [
            "press_pt",
            "press_vlv_cmd",
            "vent_vlv_cmd",
        ]
        self.subscribe(channels_to_delete)
        sy.sleep(2)
        for ch in channels_to_delete:
            try:
                console.channels.delete(ch)

                # Not getting an error immediately does not mean
                # the channel was deleted. Query the core directly.
                try:
                    client.channels.retrieve(ch)
                    self.log(f"'{ch}' still exists on core (delete was blocked)")
                except Exception:
                    self.fail(f"Channel '{ch}' improperly deleted.")

            except RuntimeError as rte:
                if "Failed to delete Channel" in str(rte):
                    self.log(f"Properly failed to delete '{ch}'")

            except Exception as e:
                self.fail(f"Unexpected error while deleting '{ch}': {e}")
