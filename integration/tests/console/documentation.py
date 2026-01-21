#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from urllib.parse import urlparse

import synnax as sy

from console.case import ConsoleCase


class Documentation(ConsoleCase):
    """Test documentation feature: opening via command palette/icon, state persistence."""

    def run(self) -> None:
        console = self.console
        docs = console.docs

        self.log("(1/3) Open documentation from command palette")
        if docs.is_open():
            docs.close()
            sy.sleep(0.3)

        assert not docs.is_open()
        docs.open_via_command_palette()
        assert docs.is_open()

        docs.wait_for_iframe_loaded()
        iframe_url = docs.get_iframe_url()
        assert "docs.synnaxlabs.com" in iframe_url
        self.log("  - Opened via command palette")

        docs.close()
        sy.sleep(0.3)

        self.log("(2/3) Open documentation from question mark icon")
        docs.open_via_question_mark_icon()
        assert docs.is_open()

        docs.wait_for_iframe_loaded()
        iframe_url = docs.get_iframe_url()
        assert "docs.synnaxlabs.com" in iframe_url
        self.log("  - Opened via question mark icon")

        self.log("(3/3) Close and reopen documentation in same place")
        docs.wait_for_iframe_loaded()
        initial_url = docs.get_iframe_url()

        docs.close()
        sy.sleep(0.5)
        assert not docs.is_open()

        docs.open_via_question_mark_icon()
        docs.wait_for_iframe_loaded()
        reopened_url = docs.get_iframe_url()

        initial_path = urlparse(initial_url).path
        reopened_path = urlparse(reopened_url).path

        self.log(f"  - Initial: {initial_path}, Reopened: {reopened_path}")
        assert (
            initial_path == reopened_path
        ), f"Path not preserved: {initial_path} != {reopened_path}"
        self.log("  - Path preserved correctly")

        docs.close()
        sy.sleep(0.3)
        self.log("All documentation tests passed!")
